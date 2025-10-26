package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Constants for cell states
const (
	Empty = " "
)

// Player identifiers and trail characters for up to 10 players
var (
	PlayerIDs  = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "A"}
	TrailChars = []string{"░", "▒", "▓", "█", "▀", "▄", "▌", "▐", "■", "□"}
)

// Direction represents a move direction
type Direction string

const (
	Up    Direction = "up"
	Down  Direction = "down"
	Left  Direction = "left"
	Right Direction = "right"
)

// Position represents a coordinate on the grid
type Position struct {
	Row int
	Col int
}

// Move represents a single move in the game
type Move struct {
	Player    string
	Direction Direction
	From      Position
	To        Position
}

// PlayerConfig holds configuration for each player
type PlayerConfig struct {
	ID          string
	Model       string
	Temperature float64
}

// GameState holds the complete game state
type GameState struct {
	Grid          [][]string
	Size          int
	NumPlayers    int
	PlayerPos     map[string]Position    // Map of player ID to position
	PlayerConfigs map[string]*PlayerConfig // Map of player ID to configuration
	ActivePlayers map[string]bool        // Track which players are still in the game
	Moves         []Move
	Visited       map[Position]bool // Track all visited positions
}

// GameStats tracks statistics across multiple games
type GameStats struct {
	PlayerWins      map[string]int // Map of player ID to win count
	Errors          int
	TotalGames      int
	ResponseTimes   []float64
	MinResponseTime float64
	MaxResponseTime float64
	AvgResponseTime float64
}

// OllamaRequest represents the request to the LLM API
type OllamaRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	Stream      bool    `json:"stream"`
	Temperature float64 `json:"temperature"`
}

// OllamaResponse represents the response from the LLM API
type OllamaResponse struct {
	Response string `json:"response"`
}

var (
	gridSize    int
	numPlayers  int
	llmURL      string
	modelName   string
	temperature float64
	maxRetries  int
	numGames    int
	debugMode   bool

	// Per-player model overrides
	player1Model string
	player2Model string
	player3Model string
	player4Model string
	player5Model string
	player6Model string
	player7Model string
	player8Model string
	player9Model string
	player10Model string
)

func init() {
	flag.IntVar(&gridSize, "size", 12, "Grid size (NxN)")
	flag.IntVar(&numPlayers, "players", 2, "Number of players (2-10)")
	flag.StringVar(&llmURL, "url", "http://localhost:11434/api/generate", "LLM API URL")
	flag.StringVar(&modelName, "model", "llama3.2", "Default model name (used if no per-player model specified)")
	flag.Float64Var(&temperature, "temp", 0.7, "Temperature for LLM")
	flag.IntVar(&maxRetries, "retries", 3, "Max retries for invalid moves")
	flag.IntVar(&numGames, "games", 1, "Number of games to play (0 for unlimited)")
	flag.BoolVar(&debugMode, "debug", false, "Enable debug mode (show prompts)")

	// Per-player model flags
	flag.StringVar(&player1Model, "model1", "", "Model for Player 1 (overrides -model)")
	flag.StringVar(&player2Model, "model2", "", "Model for Player 2 (overrides -model)")
	flag.StringVar(&player3Model, "model3", "", "Model for Player 3 (overrides -model)")
	flag.StringVar(&player4Model, "model4", "", "Model for Player 4 (overrides -model)")
	flag.StringVar(&player5Model, "model5", "", "Model for Player 5 (overrides -model)")
	flag.StringVar(&player6Model, "model6", "", "Model for Player 6 (overrides -model)")
	flag.StringVar(&player7Model, "model7", "", "Model for Player 7 (overrides -model)")
	flag.StringVar(&player8Model, "model8", "", "Model for Player 8 (overrides -model)")
	flag.StringVar(&player9Model, "model9", "", "Model for Player 9 (overrides -model)")
	flag.StringVar(&player10Model, "model10", "", "Model for Player 10 (overrides -model)")
}

// getPlayerModel returns the model for a specific player index (0-based)
func getPlayerModel(playerIndex int) string {
	playerModels := []string{
		player1Model, player2Model, player3Model, player4Model, player5Model,
		player6Model, player7Model, player8Model, player9Model, player10Model,
	}

	if playerIndex < 0 || playerIndex >= len(playerModels) {
		return modelName
	}

	// Return per-player model if specified, otherwise use default
	if playerModels[playerIndex] != "" {
		return playerModels[playerIndex]
	}
	return modelName
}

func main() {
	flag.Parse()

	// Validate number of players
	if numPlayers < 2 || numPlayers > 10 {
		fmt.Printf("Error: Number of players must be between 2 and 10 (got %d)\n", numPlayers)
		return
	}

	fmt.Println("🐍 Welcome to LLM Snakes Game! 🐍")
	fmt.Printf("Grid Size: %dx%d\n", gridSize, gridSize)
	fmt.Printf("Players: %d\n", numPlayers)

	// Display model configuration
	fmt.Println("Models:")
	for i := 0; i < numPlayers; i++ {
		playerID := PlayerIDs[i]
		model := getPlayerModel(i)
		fmt.Printf("  Player %s: %s\n", playerID, model)
	}

	fmt.Printf("API URL: %s\n\n", llmURL)

	stats := &GameStats{
		PlayerWins:      make(map[string]int),
		ResponseTimes:   make([]float64, 0),
		MinResponseTime: 999999,
		MaxResponseTime: 0,
	}

	gameCount := 0
	for {
		if numGames > 0 && gameCount >= numGames {
			break
		}

		gameCount++
		fmt.Printf("\n========== Game %d ==========\n", gameCount)

		result := PlayGame(gameCount)

		// Update statistics
		stats.TotalGames++
		if result == "error" {
			stats.Errors++
		} else if result != "" {
			stats.PlayerWins[result]++
		}

		// Display current statistics
		if numGames != 1 {
			DisplayStats(stats)
		}
	}

	if numGames != 1 {
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Println("Final Statistics:")
		DisplayStats(stats)
	}
}

// InitGame creates a new game state with random starting positions
func InitGame() *GameState {
	game := &GameState{
		Size:          gridSize,
		NumPlayers:    numPlayers,
		Grid:          make([][]string, gridSize),
		PlayerPos:     make(map[string]Position),
		PlayerConfigs: make(map[string]*PlayerConfig),
		ActivePlayers: make(map[string]bool),
		Visited:       make(map[Position]bool),
		Moves:         make([]Move, 0),
	}

	// Initialize player configurations
	for i := 0; i < numPlayers; i++ {
		playerID := PlayerIDs[i]
		game.PlayerConfigs[playerID] = &PlayerConfig{
			ID:          playerID,
			Model:       getPlayerModel(i),
			Temperature: temperature,
		}
	}

	// Initialize empty grid
	for i := 0; i < gridSize; i++ {
		game.Grid[i] = make([]string, gridSize)
		for j := 0; j < gridSize; j++ {
			game.Grid[i][j] = Empty
		}
	}

	for i := 0; i < numPlayers; i++ {
		playerID := PlayerIDs[i]
		var pos Position

		// Keep trying positions until we find one that's far enough from all existing players
		maxAttempts := 1000
		for attempt := 0; attempt < maxAttempts; attempt++ {
			pos = Position{
				Row: rand.Intn(gridSize),
				Col: rand.Intn(gridSize),
			}

			// Check if position is far enough from all existing players
			tooClose := false
			for existingPlayer, existingPos := range game.PlayerPos {
				_ = existingPlayer
				rowDiff := abs(pos.Row - existingPos.Row)
				colDiff := abs(pos.Col - existingPos.Col)
				if rowDiff+colDiff < 3 {
					tooClose = true
					break
				}
			}

			if !tooClose {
				break
			}
		}

		game.PlayerPos[playerID] = pos
		game.ActivePlayers[playerID] = true
		game.Grid[pos.Row][pos.Col] = playerID
		game.Visited[pos] = true
	}

	return game
}

// PlayGame runs a single game and returns the winner
func PlayGame(gameNumber int) string {
	game := InitGame()

	fmt.Printf("\nGame %d. Starting positions:\n", gameNumber)
	for i := 0; i < game.NumPlayers; i++ {
		playerID := PlayerIDs[i]
		pos := game.PlayerPos[playerID]
		fmt.Printf("Player %s: (%d, %d)\n", playerID, pos.Row, pos.Col)
	}
	fmt.Println()

	DisplayBoard(game)

	currentPlayerIndex := 0
	moveCount := 0

	for {
		// Find next active player
		for attempts := 0; attempts < game.NumPlayers; attempts++ {
			playerID := PlayerIDs[currentPlayerIndex]
			if game.ActivePlayers[playerID] {
				break
			}
			currentPlayerIndex = (currentPlayerIndex + 1) % game.NumPlayers
		}

		currentPlayer := PlayerIDs[currentPlayerIndex]

		// Check if only one player remains
		activeCount := 0
		var lastActivePlayer string
		for i := 0; i < game.NumPlayers; i++ {
			playerID := PlayerIDs[i]
			if game.ActivePlayers[playerID] {
				activeCount++
				lastActivePlayer = playerID
			}
		}

		if activeCount <= 1 {
			if activeCount == 1 {
				fmt.Printf("\n🎉 Player %s wins! All other players have been eliminated.\n", lastActivePlayer)
				return lastActivePlayer
			}
			fmt.Println("\n🤝 Draw! All players eliminated simultaneously.")
			return ""
		}

		moveCount++
		fmt.Printf("\n--- Move %d: Player %s's turn ---\n", moveCount, currentPlayer)

		// Get valid moves for current player
		validMoves := GetValidMoves(game, currentPlayer)

		if len(validMoves) == 0 {
			// Current player has no valid moves - they're eliminated
			game.ActivePlayers[currentPlayer] = false
			fmt.Printf("❌ Player %s is eliminated (no valid moves)\n", currentPlayer)

			// Move to next player
			currentPlayerIndex = (currentPlayerIndex + 1) % game.NumPlayers
			continue
		}

		// Get move from LLM
		direction, responseTime, err := GetLLMMove(game, currentPlayer, validMoves)

		if err != nil {
			fmt.Printf("❌ Error getting move from LLM: %v\n", err)
			return "error"
		}

		fmt.Printf("Player %s chose: %s (%.2fs)\n", currentPlayer, direction, responseTime)

		// Make the move
		MakeMove(game, currentPlayer, direction)

		DisplayBoard(game)

		// Move to next player
		currentPlayerIndex = (currentPlayerIndex + 1) % game.NumPlayers
	}
}

// GetValidMoves returns all valid directions for a player
func GetValidMoves(game *GameState, player string) []Direction {
	currentPos := game.PlayerPos[player]

	validMoves := make([]Direction, 0)

	// Check each direction
	directions := []struct {
		dir    Direction
		newPos Position
	}{
		{Up, Position{currentPos.Row - 1, currentPos.Col}},
		{Down, Position{currentPos.Row + 1, currentPos.Col}},
		{Left, Position{currentPos.Row, currentPos.Col - 1}},
		{Right, Position{currentPos.Row, currentPos.Col + 1}},
	}

	for _, d := range directions {
		if IsValidMove(game, d.newPos) {
			validMoves = append(validMoves, d.dir)
		}
	}

	return validMoves
}

// IsValidMove checks if a position is valid (in bounds and not visited)
func IsValidMove(game *GameState, pos Position) bool {
	// Check bounds
	if pos.Row < 0 || pos.Row >= game.Size || pos.Col < 0 || pos.Col >= game.Size {
		return false
	}

	// Check if already visited
	if game.Visited[pos] {
		return false
	}

	return true
}

// MakeMove executes a move for a player
func MakeMove(game *GameState, player string, direction Direction) {
	// Get player index to find trail character
	var playerIndex int
	for i := 0; i < game.NumPlayers; i++ {
		if PlayerIDs[i] == player {
			playerIndex = i
			break
		}
	}
	trailChar := TrailChars[playerIndex]

	// Get current position
	currentPos := game.PlayerPos[player]
	oldPos := currentPos

	// Calculate new position
	newPos := currentPos
	switch direction {
	case Up:
		newPos.Row--
	case Down:
		newPos.Row++
	case Left:
		newPos.Col--
	case Right:
		newPos.Col++
	}

	// Mark old position as trail
	game.Grid[oldPos.Row][oldPos.Col] = trailChar

	// Update player position
	game.PlayerPos[player] = newPos
	game.Grid[newPos.Row][newPos.Col] = player
	game.Visited[newPos] = true

	// Record the move
	move := Move{
		Player:    player,
		Direction: direction,
		From:      oldPos,
		To:        newPos,
	}
	game.Moves = append(game.Moves, move)
}

// DisplayBoard shows the current game state
func DisplayBoard(game *GameState) {
	fmt.Println()

	// Top border with column numbers
	fmt.Print("    ")
	for col := 0; col < game.Size; col++ {
		fmt.Printf("%2d  ", col)
	}
	fmt.Println()

	fmt.Print("   ┌")
	for col := 0; col < game.Size; col++ {
		fmt.Print("───")
		if col < game.Size-1 {
			fmt.Print("┬")
		}
	}
	fmt.Println("┐")

	// Grid rows
	for row := 0; row < game.Size; row++ {
		fmt.Printf("%2d │", row)
		for col := 0; col < game.Size; col++ {
			fmt.Printf(" %s │", game.Grid[row][col])
		}
		fmt.Println()

		// Row separator
		if row < game.Size-1 {
			fmt.Print("   ├")
			for col := 0; col < game.Size; col++ {
				fmt.Print("───")
				if col < game.Size-1 {
					fmt.Print("┼")
				}
			}
			fmt.Println("┤")
		}
	}

	// Bottom border
	fmt.Print("   └")
	for col := 0; col < game.Size; col++ {
		fmt.Print("───")
		if col < game.Size-1 {
			fmt.Print("┴")
		}
	}
	fmt.Println("┘")

	// Legend
	fmt.Print("\nLegend: ")
	for i := 0; i < game.NumPlayers; i++ {
		playerID := PlayerIDs[i]
		trailChar := TrailChars[i]
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Printf("%s=Player%s %s=Trail", playerID, playerID, trailChar)
	}
	fmt.Println()
}

// GetLLMMove gets a move from the LLM
func GetLLMMove(game *GameState, player string, validMoves []Direction) (Direction, float64, error) {
	prompt := BuildPrompt(game, player, validMoves)

	if debugMode {
		fmt.Println("\n=== PROMPT ===")
		fmt.Println(prompt)
		fmt.Println("=== END PROMPT ===\n")
	}

	playerConfig := game.PlayerConfigs[player]

	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			fmt.Printf("Retry %d/%d...\n", retry, maxRetries)
		}

		start := time.Now()
		response, err := CallLLM(prompt, playerConfig)
		responseTime := time.Since(start).Seconds()

		if err != nil {
			return "", 0, err
		}

		direction, err := ParseDirection(response, validMoves)
		if err == nil {
			return direction, responseTime, nil
		}

		fmt.Printf("Invalid response: %s (Error: %v)\n", response, err)
		prompt = prompt + fmt.Sprintf("\n\nYour previous response '%s' was invalid. Please respond with exactly one word: %s",
			response, formatValidMoves(validMoves))
	}

	return "", 0, fmt.Errorf("max retries exceeded")
}

// BuildPrompt creates the prompt for the LLM
func BuildPrompt(game *GameState, player string, validMoves []Direction) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("You are playing a Snakes game as Player %s.\n\n", player))

	buf.WriteString("GAME RULES:\n")
	if game.NumPlayers == 2 {
		buf.WriteString("- This is a 2-player grid-based game\n")
	} else {
		buf.WriteString(fmt.Sprintf("- This is a %d-player grid-based game\n", game.NumPlayers))
	}
	buf.WriteString("- Each player moves one cell at a time: up, down, left, or right\n")
	buf.WriteString("- Each cell you visit becomes part of your trail and can NEVER be visited again by anyone\n")
	buf.WriteString("- You LOSE if you have no valid moves (all adjacent cells are visited or out of bounds)\n")
	buf.WriteString("- Your goal: survive longer than your opponents\n\n")

	// Move history (limit to last 20 moves to keep prompt manageable)
	if len(game.Moves) > 0 {
		buf.WriteString("RECENT MOVE HISTORY:\n")
		startIdx := 0
		if len(game.Moves) > 20 {
			startIdx = len(game.Moves) - 20
		}
		for i := startIdx; i < len(game.Moves); i++ {
			move := game.Moves[i]
			buf.WriteString(fmt.Sprintf("%d. Player %s moved %s from (%d,%d) to (%d,%d)\n",
				i+1, move.Player, move.Direction, move.From.Row, move.From.Col, move.To.Row, move.To.Col))
		}
		buf.WriteString("\n")
	}

	// Current positions
	buf.WriteString("CURRENT POSITIONS:\n")
	buf.WriteString(fmt.Sprintf("- You (Player %s): (%d, %d)\n", player,
		game.PlayerPos[player].Row, game.PlayerPos[player].Col))

	// List all opponents
	for i := 0; i < game.NumPlayers; i++ {
		opponentID := PlayerIDs[i]
		if opponentID != player && game.ActivePlayers[opponentID] {
			pos := game.PlayerPos[opponentID]
			buf.WriteString(fmt.Sprintf("- Player %s: (%d, %d)\n", opponentID, pos.Row, pos.Col))
		} else if opponentID != player && !game.ActivePlayers[opponentID] {
			buf.WriteString(fmt.Sprintf("- Player %s: ELIMINATED\n", opponentID))
		}
	}
	buf.WriteString("\n")

	// Current board
	buf.WriteString("CURRENT BOARD:\n")
	buf.WriteString(formatBoardForPrompt(game))
	buf.WriteString("\n")

	// Valid moves with deep look-ahead analysis
	buf.WriteString("YOUR VALID MOVES (with deep strategic analysis):\n")
	if len(validMoves) == 0 {
		buf.WriteString("NONE - You lose!\n")
	} else {
		// Evaluate all moves and sort by score
		evaluations := make([]MoveEvaluation, 0, len(validMoves))
		for _, dir := range validMoves {
			eval := evaluateMove(game, getPlayerPos(game, player), dir)
			evaluations = append(evaluations, eval)
		}

		// Sort by score (descending)
		for i := 0; i < len(evaluations)-1; i++ {
			for j := i + 1; j < len(evaluations); j++ {
				if evaluations[j].TotalScore > evaluations[i].TotalScore {
					evaluations[i], evaluations[j] = evaluations[j], evaluations[i]
				}
			}
		}

		// Display in ranked order
		for rank, eval := range evaluations {
			rankEmoji := "  "
			if rank == 0 {
				rankEmoji = "⭐"
			} else if rank == len(evaluations)-1 && len(evaluations) > 1 {
				rankEmoji = "⚠️ "
			}
			buf.WriteString(fmt.Sprintf("%s %s → (%d,%d) | Score: %.1f | %s\n",
				rankEmoji,
				strings.ToUpper(string(eval.Direction)),
				eval.NewPos.Row, eval.NewPos.Col,
				eval.TotalScore,
				eval.SafetyLevel))
			buf.WriteString(fmt.Sprintf("   ├─ Next moves: %d | Territory: %d cells | Future mobility: %.1f\n",
				eval.ImmediateMoves,
				eval.ReachableTerritory,
				eval.AvgDepthMobility))
		}
	}
	buf.WriteString("\n")

	// Blocked moves
	blockedMoves := getBlockedMoves(game, player, validMoves)
	if len(blockedMoves) > 0 {
		buf.WriteString("BLOCKED MOVES:\n")
		for dir, reason := range blockedMoves {
			buf.WriteString(fmt.Sprintf("⛔ %s - %s\n", strings.ToUpper(string(dir)), reason))
		}
		buf.WriteString("\n")
	}

	// Strategy hints
	buf.WriteString("CRITICAL STRATEGY:\n")
	buf.WriteString("⭐ The moves are RANKED BY SCORE - higher score = better long-term survival\n")
	buf.WriteString("⭐ The top-ranked move (⭐) is calculated to give you the best chance to win\n")
	buf.WriteString("⭐ STRONGLY PREFER moves marked EXCELLENT or GOOD\n\n")
	buf.WriteString("Key Metrics Explained:\n")
	buf.WriteString("• Score: Overall quality (immediate + future mobility + territory control)\n")
	buf.WriteString("• Next moves: Options available after this move (0 = instant death next turn!)\n")
	buf.WriteString("• Territory: Total reachable space from this position (higher = more room to maneuver)\n")
	buf.WriteString("• Future mobility: Average options 2-3 moves ahead (higher = better long-term position)\n\n")
	buf.WriteString("Safety Ratings:\n")
	buf.WriteString("• EXCELLENT: Large territory + multiple options = strong survival chance\n")
	buf.WriteString("• GOOD: Decent space and mobility = reasonable position\n")
	buf.WriteString("• MODERATE: Limited but viable = be cautious\n")
	buf.WriteString("• RISKY: Very limited options = may trap yourself soon\n")
	buf.WriteString("• DANGEROUS: Poor position = avoid unless it's your only choice\n")
	buf.WriteString("• DEATH TRAP: 0 next moves = you'll lose on the next turn! NEVER choose this!\n\n")

	// Final instruction
	buf.WriteString("RESPOND WITH EXACTLY ONE WORD - YOUR CHOSEN DIRECTION:\n")
	buf.WriteString(fmt.Sprintf("Valid responses: %s\n", formatValidMoves(validMoves)))
	buf.WriteString("Do NOT include any explanation, punctuation, or other text.\n")
	buf.WriteString("Just respond with: up, down, left, or right\n")

	return buf.String()
}

// CallLLM makes the HTTP request to the LLM API
func CallLLM(prompt string, playerConfig *PlayerConfig) (string, error) {
	reqBody := OllamaRequest{
		Model:       playerConfig.Model,
		Prompt:      prompt,
		Stream:      false,
		Temperature: playerConfig.Temperature,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(llmURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", err
	}

	return strings.TrimSpace(ollamaResp.Response), nil
}

// ParseDirection extracts and validates a direction from the LLM response
func ParseDirection(response string, validMoves []Direction) (Direction, error) {
	response = strings.ToLower(strings.TrimSpace(response))

	// Try exact match first
	for _, dir := range validMoves {
		if response == string(dir) {
			return dir, nil
		}
	}

	// Try to extract direction from response using regex
	pattern := `\b(up|down|left|right)\b`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(response)

	if len(matches) > 0 {
		dir := Direction(matches[1])
		// Verify it's a valid move
		for _, validDir := range validMoves {
			if dir == validDir {
				return dir, nil
			}
		}
		return "", fmt.Errorf("direction '%s' is not valid", dir)
	}

	return "", fmt.Errorf("could not parse direction from response")
}

// DisplayStats shows game statistics
func DisplayStats(stats *GameStats) {
	fmt.Println("\n" + strings.Repeat("-", 40))
	fmt.Printf("Games Played: %d\n", stats.TotalGames)

	// Display wins for each player
	for i := 0; i < numPlayers; i++ {
		playerID := PlayerIDs[i]
		wins := stats.PlayerWins[playerID]
		percentage := 0.0
		if stats.TotalGames > 0 {
			percentage = float64(wins) / float64(stats.TotalGames) * 100
		}
		fmt.Printf("Player %s Wins: %d (%.1f%%)\n", playerID, wins, percentage)
	}

	fmt.Printf("Errors: %d\n", stats.Errors)
	fmt.Println(strings.Repeat("-", 40))
}

// Helper functions

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func getPlayerPos(game *GameState, player string) Position {
	return game.PlayerPos[player]
}

func getNewPosition(pos Position, dir Direction) Position {
	newPos := pos
	switch dir {
	case Up:
		newPos.Row--
	case Down:
		newPos.Row++
	case Left:
		newPos.Col--
	case Right:
		newPos.Col++
	}
	return newPos
}

func getBlockedMoves(game *GameState, player string, validMoves []Direction) map[Direction]string {
	blocked := make(map[Direction]string)
	currentPos := getPlayerPos(game, player)

	allDirs := []Direction{Up, Down, Left, Right}
	validMap := make(map[Direction]bool)
	for _, dir := range validMoves {
		validMap[dir] = true
	}

	for _, dir := range allDirs {
		if !validMap[dir] {
			newPos := getNewPosition(currentPos, dir)

			// Check why it's blocked
			if newPos.Row < 0 || newPos.Row >= game.Size || newPos.Col < 0 || newPos.Col >= game.Size {
				blocked[dir] = "out of bounds"
			} else if game.Visited[newPos] {
				blocked[dir] = "already visited"
			}
		}
	}

	return blocked
}

func formatValidMoves(moves []Direction) string {
	strs := make([]string, len(moves))
	for i, m := range moves {
		strs[i] = string(m)
	}
	return strings.Join(strs, ", ")
}

func formatBoardForPrompt(game *GameState) string {
	var buf bytes.Buffer

	// Column numbers
	buf.WriteString("    ")
	for col := 0; col < game.Size; col++ {
		buf.WriteString(fmt.Sprintf("%2d ", col))
	}
	buf.WriteString("\n")

	for row := 0; row < game.Size; row++ {
		buf.WriteString(fmt.Sprintf("%2d |", row))
		for col := 0; col < game.Size; col++ {
			cell := game.Grid[row][col]
			buf.WriteString(fmt.Sprintf(" %s |", cell))
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

func countAvailableMoves(game *GameState, pos Position) int {
	count := 0
	directions := []Position{
		{pos.Row - 1, pos.Col}, // up
		{pos.Row + 1, pos.Col}, // down
		{pos.Row, pos.Col - 1}, // left
		{pos.Row, pos.Col + 1}, // right
	}

	for _, newPos := range directions {
		if IsValidMove(game, newPos) {
			count++
		}
	}

	return count
}

// MoveEvaluation contains detailed evaluation of a potential move
type MoveEvaluation struct {
	Direction          Direction
	NewPos             Position
	ImmediateMoves     int     // Moves available from next position
	ReachableTerritory int     // Total reachable cells (via flood fill)
	AvgDepthMobility   float64 // Average mobility 2-3 moves ahead
	DistanceFromCenter float64 // Distance from board center (prefer center)
	TotalScore         float64 // Overall score
	SafetyLevel        string
}

// evaluateMove performs deep analysis of a move
func evaluateMove(game *GameState, currentPos Position, dir Direction) MoveEvaluation {
	newPos := getNewPosition(currentPos, dir)
	eval := MoveEvaluation{
		Direction: dir,
		NewPos:    newPos,
	}

	// Create a simulated game state after this move
	simGame := simulateMove(game, newPos)

	// 1. Immediate moves (depth 1)
	eval.ImmediateMoves = countAvailableMoves(simGame, newPos)

	// 2. Reachable territory (flood fill)
	eval.ReachableTerritory = countReachableTerritory(simGame, newPos)

	// 3. Average mobility at depth 2-3
	eval.AvgDepthMobility = calculateDepthMobility(simGame, newPos, 2)

	// 4. Distance from center (prefer center positions)
	centerRow := float64(game.Size) / 2.0
	centerCol := float64(game.Size) / 2.0
	eval.DistanceFromCenter = calculateDistance(float64(newPos.Row), float64(newPos.Col), centerRow, centerCol)

	// Calculate total score (weighted combination)
	eval.TotalScore = calculateMoveScore(eval, game.Size)

	// Determine safety level based on multiple factors
	eval.SafetyLevel = determineSafetyLevel(eval)

	return eval
}

// simulateMove creates a copy of game state with a move applied
func simulateMove(game *GameState, to Position) *GameState {
	simGame := &GameState{
		Size:    game.Size,
		Visited: make(map[Position]bool),
	}

	// Copy visited positions
	for pos := range game.Visited {
		simGame.Visited[pos] = true
	}

	// Mark the new position as visited
	simGame.Visited[to] = true

	return simGame
}

// countReachableTerritory uses flood fill to count all reachable cells
func countReachableTerritory(game *GameState, start Position) int {
	visited := make(map[Position]bool)
	queue := []Position{start}
	visited[start] = true
	count := 1

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Check all 4 directions
		directions := []Position{
			{current.Row - 1, current.Col},
			{current.Row + 1, current.Col},
			{current.Row, current.Col - 1},
			{current.Row, current.Col + 1},
		}

		for _, next := range directions {
			if !visited[next] && IsValidMove(game, next) {
				visited[next] = true
				queue = append(queue, next)
				count++
			}
		}
	}

	return count
}

// calculateDepthMobility calculates average mobility at future depths
func calculateDepthMobility(game *GameState, pos Position, maxDepth int) float64 {
	if maxDepth <= 0 {
		return 0
	}

	// Get available moves from current position
	moves := getAvailablePositions(game, pos)
	if len(moves) == 0 {
		return 0
	}

	totalMobility := 0.0
	for _, nextPos := range moves {
		// Create simulation for this path
		simGame := simulateMove(game, nextPos)
		// Count moves from this position
		mobilityAtNext := float64(countAvailableMoves(simGame, nextPos))
		totalMobility += mobilityAtNext

		// Recursive check at next depth (but limit to avoid performance issues)
		if maxDepth > 1 {
			deeperMobility := calculateDepthMobility(simGame, nextPos, maxDepth-1)
			totalMobility += deeperMobility * 0.5 // Weight future moves less
		}
	}

	return totalMobility / float64(len(moves))
}

// getAvailablePositions returns all valid positions reachable from pos
func getAvailablePositions(game *GameState, pos Position) []Position {
	positions := []Position{}
	directions := []Position{
		{pos.Row - 1, pos.Col},
		{pos.Row + 1, pos.Col},
		{pos.Row, pos.Col - 1},
		{pos.Row, pos.Col + 1},
	}

	for _, newPos := range directions {
		if IsValidMove(game, newPos) {
			positions = append(positions, newPos)
		}
	}

	return positions
}

// calculateDistance calculates Euclidean distance
func calculateDistance(x1, y1, x2, y2 float64) float64 {
	dx := x1 - x2
	dy := y1 - y2
	return (dx*dx + dy*dy) // Skip sqrt for performance, we only need relative comparison
}

// calculateMoveScore computes weighted score for a move
func calculateMoveScore(eval MoveEvaluation, boardSize int) float64 {
	score := 0.0

	// Territory is most important (can I control space?)
	score += float64(eval.ReachableTerritory) * 2.0

	// Deep mobility is very important (will I have options later?)
	score += eval.AvgDepthMobility * 3.0

	// Immediate moves matter (don't trap yourself next turn)
	score += float64(eval.ImmediateMoves) * 5.0

	// Slight preference for center positions (avoid corners/edges)
	maxDist := float64(boardSize * boardSize)
	centerScore := (maxDist - eval.DistanceFromCenter) / maxDist
	score += centerScore * 1.0

	return score
}

// determineSafetyLevel categorizes move safety
func determineSafetyLevel(eval MoveEvaluation) string {
	// Consider multiple factors
	if eval.ImmediateMoves == 0 {
		return "DEATH TRAP"
	}

	if eval.ReachableTerritory >= 20 && eval.ImmediateMoves >= 3 {
		return "EXCELLENT"
	}

	if eval.ReachableTerritory >= 12 && eval.ImmediateMoves >= 2 {
		return "GOOD"
	}

	if eval.ImmediateMoves >= 2 && eval.ReachableTerritory >= 6 {
		return "MODERATE"
	}

	if eval.ImmediateMoves == 1 || eval.ReachableTerritory < 5 {
		return "RISKY"
	}

	return "DANGEROUS"
}
