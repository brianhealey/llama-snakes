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

// Constants for players and cell states
const (
	Player1   = "1"
	Player2   = "2"
	Empty     = " "
	Trail1    = "░" // Light shade for Player 1's trail
	Trail2    = "▓" // Dark shade for Player 2's trail
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

// GameState holds the complete game state
type GameState struct {
	Grid      [][]string
	Size      int
	Player1Pos Position
	Player2Pos Position
	Moves     []Move
	Visited   map[Position]bool // Track all visited positions
}

// GameStats tracks statistics across multiple games
type GameStats struct {
	Player1Wins    int
	Player2Wins    int
	Errors         int
	TotalGames     int
	ResponseTimes  []float64
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
	llmURL      string
	modelName   string
	temperature float64
	maxRetries  int
	numGames    int
	debugMode   bool
)

func init() {
	flag.IntVar(&gridSize, "size", 12, "Grid size (NxN)")
	flag.StringVar(&llmURL, "url", "http://localhost:11434/api/generate", "LLM API URL")
	flag.StringVar(&modelName, "model", "llama3.2", "Model name")
	flag.Float64Var(&temperature, "temp", 0.7, "Temperature for LLM")
	flag.IntVar(&maxRetries, "retries", 3, "Max retries for invalid moves")
	flag.IntVar(&numGames, "games", 1, "Number of games to play (0 for unlimited)")
	flag.BoolVar(&debugMode, "debug", false, "Enable debug mode (show prompts)")
}

func main() {
	flag.Parse()

	fmt.Println("🐍 Welcome to LLM Snakes Game! 🐍")
	fmt.Printf("Grid Size: %dx%d\n", gridSize, gridSize)
	fmt.Printf("Model: %s\n", modelName)
	fmt.Printf("API URL: %s\n\n", llmURL)

	stats := &GameStats{
		ResponseTimes: make([]float64, 0),
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
		switch result {
		case Player1:
			stats.Player1Wins++
		case Player2:
			stats.Player2Wins++
		case "error":
			stats.Errors++
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
		Size:    gridSize,
		Grid:    make([][]string, gridSize),
		Visited: make(map[Position]bool),
		Moves:   make([]Move, 0),
	}

	// Initialize empty grid
	for i := 0; i < gridSize; i++ {
		game.Grid[i] = make([]string, gridSize)
		for j := 0; j < gridSize; j++ {
			game.Grid[i][j] = Empty
		}
	}

	// Random starting positions (ensuring they're not adjacent)
	rand.Seed(time.Now().UnixNano())

	// Player 1 starting position
	game.Player1Pos = Position{
		Row: rand.Intn(gridSize),
		Col: rand.Intn(gridSize),
	}
	game.Grid[game.Player1Pos.Row][game.Player1Pos.Col] = Player1
	game.Visited[game.Player1Pos] = true

	// Player 2 starting position (ensure it's at least 3 cells away)
	for {
		game.Player2Pos = Position{
			Row: rand.Intn(gridSize),
			Col: rand.Intn(gridSize),
		}

		// Check if far enough away
		rowDiff := abs(game.Player1Pos.Row - game.Player2Pos.Row)
		colDiff := abs(game.Player1Pos.Col - game.Player2Pos.Col)

		if rowDiff+colDiff >= 3 {
			break
		}
	}

	game.Grid[game.Player2Pos.Row][game.Player2Pos.Col] = Player2
	game.Visited[game.Player2Pos] = true

	return game
}

// PlayGame runs a single game and returns the winner
func PlayGame(gameNumber int) string {
	game := InitGame()

	fmt.Println("\nStarting positions:")
	fmt.Printf("Player 1: (%d, %d)\n", game.Player1Pos.Row, game.Player1Pos.Col)
	fmt.Printf("Player 2: (%d, %d)\n\n", game.Player2Pos.Row, game.Player2Pos.Col)

	DisplayBoard(game)

	currentPlayer := Player1
	moveCount := 0

	for {
		moveCount++
		fmt.Printf("\n--- Move %d: Player %s's turn ---\n", moveCount, currentPlayer)

		// Get valid moves for current player
		validMoves := GetValidMoves(game, currentPlayer)

		if len(validMoves) == 0 {
			// Current player has no valid moves - they lose
			winner := Player2
			if currentPlayer == Player2 {
				winner = Player1
			}
			fmt.Printf("\n🎉 Player %s wins! Player %s has no valid moves.\n", winner, currentPlayer)
			return winner
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

		// Switch players
		if currentPlayer == Player1 {
			currentPlayer = Player2
		} else {
			currentPlayer = Player1
		}
	}
}

// GetValidMoves returns all valid directions for a player
func GetValidMoves(game *GameState, player string) []Direction {
	var currentPos Position
	if player == Player1 {
		currentPos = game.Player1Pos
	} else {
		currentPos = game.Player2Pos
	}

	validMoves := make([]Direction, 0)

	// Check each direction
	directions := []struct {
		dir Direction
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
	var currentPos *Position
	var trailChar string

	if player == Player1 {
		currentPos = &game.Player1Pos
		trailChar = Trail1
	} else {
		currentPos = &game.Player2Pos
		trailChar = Trail2
	}

	// Save old position
	oldPos := *currentPos

	// Calculate new position
	newPos := *currentPos
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
	*currentPos = newPos
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
		fmt.Printf("%2d ", col)
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

	fmt.Printf("\nLegend: 1=Player1  2=Player2  %s=Player1 Trail  %s=Player2 Trail\n", Trail1, Trail2)
}

// GetLLMMove gets a move from the LLM
func GetLLMMove(game *GameState, player string, validMoves []Direction) (Direction, float64, error) {
	prompt := BuildPrompt(game, player, validMoves)

	if debugMode {
		fmt.Println("\n=== PROMPT ===")
		fmt.Println(prompt)
		fmt.Println("=== END PROMPT ===\n")
	}

	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			fmt.Printf("Retry %d/%d...\n", retry, maxRetries)
		}

		start := time.Now()
		response, err := CallLLM(prompt)
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

	opponent := Player2
	if player == Player2 {
		opponent = Player1
	}

	buf.WriteString(fmt.Sprintf("You are playing a Snakes game as Player %s.\n\n", player))

	buf.WriteString("GAME RULES:\n")
	buf.WriteString("- This is a 2-player grid-based game\n")
	buf.WriteString("- Each player moves one cell at a time: up, down, left, or right\n")
	buf.WriteString("- Each cell you visit becomes part of your trail and can NEVER be visited again by anyone\n")
	buf.WriteString("- You LOSE if you have no valid moves (all adjacent cells are visited or out of bounds)\n")
	buf.WriteString("- Your goal: survive longer than your opponent\n\n")

	// Move history
	if len(game.Moves) > 0 {
		buf.WriteString("MOVE HISTORY:\n")
		for i, move := range game.Moves {
			buf.WriteString(fmt.Sprintf("%d. Player %s moved %s from (%d,%d) to (%d,%d)\n",
				i+1, move.Player, move.Direction, move.From.Row, move.From.Col, move.To.Row, move.To.Col))
		}
		buf.WriteString("\n")
	}

	// Current positions
	buf.WriteString("CURRENT POSITIONS:\n")
	buf.WriteString(fmt.Sprintf("- You (Player %s): (%d, %d)\n", player,
		getPlayerPos(game, player).Row, getPlayerPos(game, player).Col))
	buf.WriteString(fmt.Sprintf("- Opponent (Player %s): (%d, %d)\n\n", opponent,
		getPlayerPos(game, opponent).Row, getPlayerPos(game, opponent).Col))

	// Current board
	buf.WriteString("CURRENT BOARD:\n")
	buf.WriteString(formatBoardForPrompt(game))
	buf.WriteString("\n")

	// Valid moves
	buf.WriteString("YOUR VALID MOVES:\n")
	if len(validMoves) == 0 {
		buf.WriteString("NONE - You lose!\n")
	} else {
		for _, dir := range validMoves {
			newPos := getNewPosition(getPlayerPos(game, player), dir)
			buf.WriteString(fmt.Sprintf("✅ %s - moves to (%d, %d)\n", strings.ToUpper(string(dir)), newPos.Row, newPos.Col))
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
	buf.WriteString("STRATEGY:\n")
	buf.WriteString("1. Try to move toward open space with many available moves\n")
	buf.WriteString("2. Avoid corners and edges when possible\n")
	buf.WriteString("3. Try to cut off your opponent's escape routes\n")
	buf.WriteString("4. Stay away from your own trail and the opponent's trail\n\n")

	// Final instruction
	buf.WriteString("RESPOND WITH EXACTLY ONE WORD - YOUR CHOSEN DIRECTION:\n")
	buf.WriteString(fmt.Sprintf("Valid responses: %s\n", formatValidMoves(validMoves)))
	buf.WriteString("Do NOT include any explanation, punctuation, or other text.\n")
	buf.WriteString("Just respond with: up, down, left, or right\n")

	return buf.String()
}

// CallLLM makes the HTTP request to the LLM API
func CallLLM(prompt string) (string, error) {
	reqBody := OllamaRequest{
		Model:       modelName,
		Prompt:      prompt,
		Stream:      false,
		Temperature: temperature,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(llmURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

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
	fmt.Printf("Player 1 Wins: %d (%.1f%%)\n", stats.Player1Wins,
		float64(stats.Player1Wins)/float64(stats.TotalGames)*100)
	fmt.Printf("Player 2 Wins: %d (%.1f%%)\n", stats.Player2Wins,
		float64(stats.Player2Wins)/float64(stats.TotalGames)*100)
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
	if player == Player1 {
		return game.Player1Pos
	}
	return game.Player2Pos
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
