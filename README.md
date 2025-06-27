# snsm - Simple Notes for Simple Man
I'm a simple man with simple need. I just needed a simple program to open my notes (in markdown) and search them with tags.

## Usage
Just start `snsm` and enter the tags you're searching for. Selecting one will open it in your favorite `$EDITOR`. 

### Features
- **Create Notes**: Press `n` to create a new note
- **Timestamps**: Use `%t` in your filename to insert the current date (format: YYYY-MM-DD)
- **Tag Support**: Add tags to your notes to easily retrieve them
- **Filtering**: Filter notes by both filename and tags
- **Simple Storage**: Just plain text files, you choose how you back it/sync it

### Nice to have in the future
- **Deep Search**: When no matches are found by tags, it will search in file contents
- **CLI options**: to be able to create new tagged note directly with a cli.

### Tagging
To tag a document just add them on the first line of your document like so:
```md
// +golang +howto
# How to use goroutine
...
```

### Directory Structure
By default, notes are stored in `~/notes/`. This directory will be created for you if it doesn't exist.

### Navigation
- Use arrow keys or vim keys to navigate through notes
- Press `enter` to open a note in your editor
- Press `q` to quit
