# LLM Snakes Game

A two-player grid-based game where LLMs play against each other. Each player moves through a grid, leaving a trail behind them. Once a cell is visited, it cannot be visited again. The game ends when a player has no valid moves remaining.

## Game Rules

- **Grid**: Configurable NxN grid (default 12x12)
- **Starting Positions**: Players start at random positions at least 3 cells apart
- **Moves**: Each turn, a player moves one cell in a direction: up, down, left, or right
- **Trail**: Every visited cell becomes part of a player's trail and is permanently blocked
- **Win Condition**: A player wins when their opponent has no valid moves
- **Loss Condition**: A player loses when they cannot move in any direction without revisiting a cell

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
# Custom grid size
./llama-snakes -size 15

# Use different LLM endpoint (Ollama/LM Studio/etc)
./llama-snakes -url http://localhost:11434/api/generate

# Specify model
./llama-snakes -model llama3.2

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

# Small grid for faster games
./llama-snakes -size 8 -games 5

# Debug a single game
./llama-snakes -debug -games 1
```

## How It Works

### Game Flow

1. **Initialization**: Two players are placed at random positions on the grid
2. **Turn-Based Play**: Players alternate taking turns
3. **LLM Decision**: Each turn, the LLM receives:
   - Complete game state and move history
   - Current board visualization
   - List of valid moves
   - Strategic hints
4. **Move Execution**: The chosen direction is validated and executed
5. **Trail Marking**: The previous position becomes part of the player's trail
6. **Win Detection**: Game ends when a player has no valid moves

### Prompt Engineering

The LLM receives comprehensive context including:
- Full move history
- Current positions of both players
- Visual board representation
- List of valid moves with target positions
- List of blocked moves with reasons
- Strategic guidance
- Clear response format instructions

### Visualization

- `1` - Player 1's current position
- `2` - Player 2's current position
- `░` - Player 1's trail (light shade)
- `▓` - Player 2's trail (dark shade)
- ` ` - Empty, visitable cells

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
- Win percentages
- Error counts

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
🐍 Welcome to LLM Snakes Game! 🐍
Grid Size: 12x12
Model: llama3.2
API URL: http://localhost:11434/api/generate

========== Game 1 ==========

Starting positions:
Player 1: (3, 8)
Player 2: (9, 2)

    0  1  2  3  4  5  6  7  8  9 10 11
   ┌──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┐
 0 │  │  │  │  │  │  │  │  │  │  │  │  │
   ├──┼──┼──┼──┼──┼──┼──┼──┼──┼──┼──┼──┤
...
 3 │  │  │  │  │  │  │  │  │ 1 │  │  │  │
...
 9 │  │  │ 2 │  │  │  │  │  │  │  │  │  │
   └──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘

--- Move 1: Player 1's turn ---
Player 1 chose: right (1.34s)
...
```

## License

MIT
