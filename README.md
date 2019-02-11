# High-level goal
Automatically move downloaded media into a Plex-known path

# Steps
- query transmission-daemon for recently completed torrents
- rule out those already completed
- if extraction needed:
    - unrar + mv to Plex-known path
- else:
    - cp to Plex-known path
