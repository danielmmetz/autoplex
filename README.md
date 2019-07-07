# autoplex
Autoplex is a tool that moves (and extracts) transmission-downloaded media for
ease of integration with Plex.

## Usage
```bash
> autoplex --help
Usage of ./autoplex:
      --dest string          destination directory for extracted files (default "/media/TV")
      --frequency duration   duration between runs (default 1m0s)
      --media-dir strings    directory in which to search for previously extracted files (de
fault [/media/TV,/media/Movies])
```

## Example
```bash
autoplex --dest /tv --media-dir /tv --media-dir /movies
```
