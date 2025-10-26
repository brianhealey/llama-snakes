# LLM Snakes Game

A multi-player grid-based game where LLMs play against each other. Each player moves through a grid, leaving a trail behind them. Once a cell is visited, it cannot be visited again. Players are eliminated when they have no valid moves remaining, and the last player standing wins.

## Game Rules

- **Players**: 2-10 players (default 2)
- **Grid**: Configurable NxN grid (default 12x12)
- **Starting Positions**: Players start at random positions at least 3 cells apart
- **Moves**: Each turn, a player moves one cell in a direction: up, down, left, or right
- **Trail**: Every visited cell becomes part of a player's trail and is permanently blocked
- **Elimination**: A player is eliminated when they have no valid moves
- **Win Condition**: The last player remaining wins the game

## Installation

```bash
# Clone the repository
cd llama-snakes-game

# Build the game
go build -o llama-snakes main.go
```

## Usage

### Basic Usage

```bash
# Run with default settings (requires Ollama running locally)
./llama-snakes
```

### Configuration Options

```bash
# Number of players (2-10)
./llama-snakes -players 4

# Custom grid size
./llama-snakes -size 15

# Use different LLM endpoint (Ollama/LM Studio/etc)
./llama-snakes -url http://localhost:11434/api/generate

# Specify default model for all players
./llama-snakes -model llama3.2

# Use different models per player (e.g., llama3.2 vs mistral)
./llama-snakes -model1 llama3.2 -model2 mistral

# Mix multiple models in a multi-player game
./llama-snakes -players 3 -model1 llama3.2 -model2 qwen2.5 -model3 gemma2

# Override only specific players (others use default)
./llama-snakes -players 4 -model llama3.2 -model2 mistral
# Player 1, 3, 4 use llama3.2; Player 2 uses mistral

# Adjust temperature for more creative/deterministic play
./llama-snakes -temp 0.5

# Play multiple games
./llama-snakes -games 10

# Enable debug mode (shows prompts sent to LLM)
./llama-snakes -debug

# Adjust max retries for invalid moves
./llama-snakes -retries 5
```

### Example Commands

```bash
# Tournament mode: 100 games with statistics
./llama-snakes -games 100 -size 12

# 4-player battle royale
./llama-snakes -players 4 -size 15

# Small grid for faster games
./llama-snakes -size 8 -games 5

# Debug a single game
./llama-snakes -debug -games 1

# 6-player tournament
./llama-snakes -players 6 -size 20 -games 50

# Model battle: Test llama3.2 vs mistral across 100 games
./llama-snakes -model1 llama3.2 -model2 mistral -games 100

# Multi-model tournament
./llama-snakes -players 4 -model1 llama3.2 -model2 mistral -model3 qwen2.5 -model4 gemma2 -games 50
```

## How It Works

### Per-Player Model Configuration

Each player can be assigned a different LLM model, allowing you to:
- Compare performance between different models (e.g., llama3.2 vs mistral)
- Run tournaments with multiple models competing simultaneously
- Test strategic capabilities across different LLM architectures

Use the `-model1`, `-model2`, etc. flags to specify models per player. Any player without a specific model will use the default model specified by `-model`.

### Game Flow

1. **Initialization**: Players are placed at random positions on the grid (at least 3 cells apart)
2. **Turn-Based Play**: Players take turns in rotation
3. **LLM Decision**: Each turn, the LLM receives:
   - Complete game state and move history
   - Current board visualization
   - List of valid moves with look-ahead analysis
   - Strategic hints to avoid self-entrapment
   - Positions of all active and eliminated players
4. **Move Execution**: The chosen direction is validated and executed
5. **Trail Marking**: The previous position becomes part of the player's trail
6. **Elimination**: Players are eliminated when they have no valid moves
7. **Victory**: Last player standing wins

### Prompt Engineering

The LLM receives comprehensive context including:
- Full move history for all players
- Current positions and status of all players (active/eliminated)
- Visual board representation
- List of valid moves with look-ahead analysis
- Future move counts for each direction (safety ratings)
- List of blocked moves with reasons
- Strategic guidance to avoid self-entrapment
- Clear response format instructions

The prompt includes a sophisticated look-ahead system that analyzes how many moves will be available after each possible move, helping the LLM avoid trapping itself.

### Visualization

- `1`, `2`, `3`, etc. - Player current positions
- `░`, `▒`, `▓`, `█`, etc. - Player trails (unique pattern per player)
- ` ` - Empty, visitable cells

Each player has a unique trail pattern to distinguish their paths on the board.

## Requirements

- Go 1.21 or higher
- Running LLM server (Ollama, LM Studio, or compatible API)

### Setting Up Ollama

```bash
# Install Ollama
curl https://ollama.ai/install.sh | sh

# Pull a model
ollama pull llama3.2

# Ollama runs automatically on localhost:11434
```

## Architecture

Based on the llama-tac-toe architecture:
- Single-file Go implementation
- Comprehensive prompt construction
- Robust move parsing with retry logic
- Statistics tracking across multiple games
- Configurable via command-line flags

## Game Statistics

When playing multiple games, the program tracks:
- Total games played
- Win counts for each player
- Win percentages for each player
- Error counts
- Response times

When using different models per player, statistics allow you to compare model performance and determine which models excel at strategic planning.

## Troubleshooting

**LLM gives invalid responses:**
- Try increasing `-retries` flag
- Adjust `-temp` to make responses more deterministic (lower values)
- Enable `-debug` to see prompts and responses

**Games are too short:**
- Increase grid size: `-size 15` or `-size 20`

**Games take too long:**
- Decrease grid size: `-size 8` or `-size 10`
- Use a faster model

## Example Output

```

     0   1   2   3   4   5   6   7   8   9  10  11  
   ┌───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┐
 0 │   │   │   │   │   │   │   │   │   │   │   │   │
   ├───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┤
 1 │   │   │   │   │   │   │   │   │   │   │   │   │
   ├───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┤
 2 │   │   │   │   │   │   │   │   │   │   │   │   │
   ├───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┤
 3 │   │   │   │   │   │   │   │   │   │ 2 │   │   │
   ├───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┤
 4 │   │   │   │   │   │   │   │   │   │ ▒ │   │   │
   ├───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┤
 5 │   │   │   │   │   │   │   │   │   │ ▒ │   │   │
   ├───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┤
 6 │   │   │   │   │   │   │   │   │   │ ▒ │   │   │
   ├───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┤
 7 │   │   │ 1 │   │   │   │   │   │   │ ▒ │ ▒ │   │
   ├───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┤
 8 │   │   │ ░ │   │   │   │   │   │   │   │   │   │
   ├───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┤
 9 │   │   │ ░ │   │   │   │   │   │   │   │   │   │
   ├───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┤
10 │   │ ░ │ ░ │   │   │   │   │   │   │   │   │   │
   ├───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┤
11 │   │ ░ │   │   │   │   │   │   │   │   │   │   │
   └───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┘

Legend: 1=Player1 ░=Trail  2=Player2 ▒=Trail

--- Move 1: Player 1's turn ---
Player 1 chose: right (1.34s)
...
```

## License

MIT
