# Sortd - Let Chaos Sort Itself Out! 🗂️

[![Build Status](https://img.shields.io/github/workflow/status/yourusername/sortd/Go)](https://github.com/yourusername/sortd/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/sortd)](https://goreportcard.com/report/github.com/yourusername/sortd)
[![Code Coverage](https://img.shields.io/codecov/c/github/yourusername/sortd)](https://codecov.io/gh/yourusername/sortd)
[![License](https://img.shields.io/github/license/yourusername/sortd)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/yourusername/sortd)](go.mod)

*Because life's too short to manually organize your Downloads folder!*

</div>

## What's This? 🤔

Sortd is a file organization tool built to tackle digital clutter for the chronically disorganized (looking at you, fellow ADHD brains! 👋). It automatically whisks away your files into neat little folders based on rules YOU define.

Think of it as that friend who loves organizing stuff while you're busy getting distracted by 17 different YouTube videos about how to be more productive.

## Features ✨

- **Set It & Forget It**: Watch directories and automatically organize files as they appear
- **Rule Your Domain**: Custom patterns to match files and send them where they belong
- **Content Detective**: Analyzes files to make smart decisions about where they should go
- **Your Choice of Interface**: CLI for the terminal lovers, GUI for the clickers
- **Workflow Wonder**: Chain together complex operations for maximum organization magic
- **No Conflicts**: Smart handling of duplicate filenames (no more "document(17).pdf")

## Show Me The Magic! 🪄

![Sortd in action](https://raw.githubusercontent.com/yourusername/sortd/main/docs/demo.gif)

*Imagine a beautiful GIF here showing files magically organizing themselves*

## Quick Start 🚀

### Installation

```bash
Clone the repository
git clone https://github.com/yourusername/sortd.git
cd sortd
```

Build the application
```bash
./build_no_tui.sh
Or use the Makefile
make build
```

Organize:
```yaml
patterns:
match: ".{jpg,png,gif}"
target: "Images/"
match: ".{doc,docx,pdf,txt}"
target: "Documents/"
match: ".{mp3,wav,flac}"
target: "Music/"
settings:
dry_run: false
create_dirs: true
confirm: false
collision: "rename"
watch_directories:
"~/Downloads"
```

One-time organization (for that dopamine hit!)
```bash
sortd organize ~/Downloads
```

Set up a watcher (for the "wow it happened automagically!" experience)
```bash
sortd watch
```

Use the GUI if you're feeling fancy
```bash
sortd gui
```

## Architecture (For The Curious) 🏗️

Sortd is built with a modular architecture that separates concerns:

- **Core Engine**: Handles file operations and rule matching
- **Watchers**: Monitors directories using fsnotify
- **Analyzers**: Detects file types and extracts metadata
- **Workflows**: Chains multiple actions together

It uses Go's concurrency model for efficient file processing, with worker pools to handle operations in parallel without overwhelming the system.

## Development Notes 🧪

This project started as a learning experience to:

1. Explore Go architecture patterns in a real-world context
2. Make all the concurrency mistakes so you don't have to!
3. Learn how to build software that doesn't crash when weird filenames appear
4. Create something actually useful for ADHD brains (including mine)

It embraces the philosophy of "accelerated mistakes" - deliberately exploring edge cases and potential issues to build intuition about robust software design.

## Contributing 🤝

Found this useful? Want to add a feature? Discovered a bug?

This started as a personal project, but if you find it helpful, contributions are welcome! Check out CONTRIBUTING.md for guidelines.

## Future Plans 🔮

See `docs/plans.md` for the full roadmap, but highlights include:

- Smarter content analysis for better auto-categorization
- Improved handling of edge cases and weird filenames
- More workflow actions and triggers
- Undo functionality (for when the robot gets it wrong)

## License 📝

[Add your chosen license here]

## Final Thought 💭

Remember: organization is not about perfection, it's about reducing friction. If this tool helps you spend less time sorting files and more time doing what you love, it's done its job!

*"Organized people are just too lazy to look for things." - Albert Einstein (probably)*
