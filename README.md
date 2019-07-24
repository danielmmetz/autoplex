# autoplex
Autoplex is a tool that moves (and extracts) transmission-downloaded media for
ease of integration with Plex.

## Usage
```bash
> autoplex --help
Usage of ./autoplex:
      --dest strings         destination directory for extracted files
      --frequency duration   duration between runs (default 1m0s)
      --src strings          source directory for downloaded files
```

## Example
```bash
autoplex --src /downloads/tv --src /downloads/movies --dest /tv --dest /movies
```
