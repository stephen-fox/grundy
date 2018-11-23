# grundy

## What is it?
Grundy crushes your non-Steam games into Steam shortcuts so you do not have to!
It is a headless service that looks for configuration files that describe your
games. When it finds a configuration file, it creates a Steam shortcut. This
design allows for portability. By backing up a few configuration files, you
can easily add your game collections to Steam when you move from one computer
to another.

## Why though?
I created this application so I could easily add my Nintendo Gamecube games
to Steam and emulate them using Dolphin. I wanted to let other people use it,
so I tried to make it as extensible as possible.

## How do I set it up?
Download an installer from the tags / releases section and run it. Grundy
starts automatically once the installation finishes - there are no extra
steps after the installation completes.

## What is the state of this thing?
It works, but it is rough around the edges. It is also not very practical for
users that want a UI.

## Adding your games to Steam
The following steps explain how to add your games to Steam. If you are feeling
adventurous, you can find example configuration files in the main settings
directory under `examples`.

#### 1. Organize your game collections
First, you should organize your games into what I call `game collections`.
A game collection is simply a directory (folder) that contains subdirectories
with your games' files. For example, if you had a collection of Nintendo
Gamecube games consisting of Metroid Prime and Pikmin, you would create the
following directory structure (the name does not need to be `gamecube-games`):
```
gamecube-games/
|
|---- Metroid Prime/
|   |
|   |---- mprime.gcm
|---- Pikmin/
    |
    |---- pikmin.gcm
```

This will let us tell Grundy where your games live. Once you have done this,
move on to the next step.

#### 2. Tell grundy where your games live
Once you have made a game collection, we will need to tell grundy about it.
Depending on your system, we will be working in the following directory
for the next steps:

- macOS and Linux: `~/.grundy` (where `~` is the path to your home directory)
- Windows: `C:\ProgramData\.grundy` (note the leading period)

You will want to copy the path to the collection. On Windows, you can do this
by opening the game collection in the Windows File Explorer and clicking on the
location bar. This will reveal the path, which will look something like:
`C:\Users\Me\Documents\My Games`.

Now open up `app.grundy.ini` in a text editor (like Notepad). You will see
a line starting with `[game_collections]`. On the next line, paste your the
game collection path and surround it with single quotes. For example, this file
should now look like this:
```ini
[settings]

[game_collections]
'C:\Users\Me\Documents\My Games'
```

#### 3. Tell grundy how the games should be launched
Grundy will need to know how your games should be launched. This information
is stored in the `launchers.grundy.ini`. Each section in this file will
represent a launcher.

Let's pretend we are using Dolphin to run the collection we added in the
previous step. Go ahead and add the following to the file:

```ini
[dolphin]
exe_path     = 'C:\Program Files\Dolphin\Dolphin.exe'
default_args = /e
```

Now when we tell grundy that our game uses the `dolphin` launcher, grundy will
make a Steam shortcut that launches the game using Dolphin with the `/e` (run
game) argument.

#### 4. Configure your games
