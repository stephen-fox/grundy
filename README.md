# grundy

![Shortcuts created by grundy](docs/images/steam-preview.png)

## What is it?
Grundy crushes your non-Steam games into Steam shortcuts so you do not have to!
It is a headless service that looks for configured game collections. It will
automatically add your games to Steam based on your configured game launchers.
Your changes are automatically discovered.

## Why?
I created this application so I could easily add my Nintendo Gamecube games
to Steam and emulate them using Dolphin. I wanted to let other people use it,
so I tried to make it as extensible as possible.

## How do I set it up?
Download the installer from the tags / releases section and run it. Grundy
starts automatically once the installation finishes - there are no extra
steps after the installation completes. It will automatically start on system
startup and login.

## Adding your games to Steam
The following steps explain how to add your games to Steam using
the application.

#### 1. Organize your game collections
First, you should organize your games into what I call `game collections`.
A game collection is a directory (folder) that contains subdirectories with
your games' files. A game collection can be stored anywhere on your computer.

For example, if you have a collection of Nintendo Gamecube games consisting
of Metroid Prime and Pikmin, you should create the following directory
structure (the name does not need to be `gamecube-games`):

Note: A trailing `/` indicates a directory (folder):
```
gamecube-games/
|
|---- Metroid Prime/
|   |
|   |---- mprime.gcm
|
|---- Pikmin/
    |
    |---- pikmin.gcm
```

This means you should create a directory named `gamecube-games` (or something
similar) and then create two directories inside named `Metroid Prime` and
`Pikmin`. These names will tell grundy how to name the shortcuts. In other
words, since you named the Metroid Prime game directory `Metroid Prime`, the
Steam shortcut will also be called `Metroid Prime`.

You would then copy the Gamecube files into their respective directories. The
Gamecube files do not need to be named in any particular manner. Just make sure
they have a file extension (usually `.gcm`).

#### 2. (Optional) Add icons and grid images
Steam allows you to set custom images in the following forms:
- Icons for use the compact "Games Details" view (e.g., 64x64 pixels)
- Grid images for use with the "Games Grid" view and "Big Picture" mode

If you would like grundy to add an icon or grid image for your shortcut, make
sure to copy your images into their respective directories. The image files
must end with the following suffixes:
- Icons:
    - `-icon.png`
    - `-icon.jpg`
- Grid images:
    - `-grid.png`
    - `-grid.jpg`

If multiple files exist with the above suffixes, grundy will pick the first one
it finds. Make sure your images conform to Steam's image requirements.

For example, if you have images for Metroid Prime and Pikmin named
`METROID DUDE.png` and `Pikmin Are Cool.png`, make sure they are renamed to
`METROID DUDE-grid.png` and `Pikmin Are Cool-grid.png`:
```
gamecube-games/
|
|---- Metroid Prime/
|   |
|   |---- mprime.gcm
|   |---- METROID DUDE-grid.png
|
|---- Pikmin/
    |
    |---- pikmin.gcm
    |---- Pikmin Are Cool-grid.png
```

#### 3. Find the main settings directory
Now we need to define some settings so grundy will add games collections
to Steam. These settings can be found in the main settings directory.

The main settings directory can be found in the following locations, depending
on your operating system:

- macOS and Linux: `~/.grundy` (where `~` is the path to your home directory)
- Windows: `C:\ProgramData\.grundy` (note the leading period)

#### 4. Tell grundy how your game launchers work
Once you have located the main settings directory, we will need to edit
`launchers.grundy.ini`. You can open this file in a text editor (such as
Notepad). Each section in this file will represent a launcher (i.e., a game
emulator such as Dolphin).

Let's pretend we are using Dolphin to run the collection we added in the
previous step. Go ahead and add the following to the file:
```ini
[dolphin]
exe_path           = C:\Program Files\Dolphin\Dolphin.exe
default_args       = /e
game_file_suffixes = .gcm
```

This tells grundy that there is a launcher named `dolphin` that uses the
executable `C:\Program Files\Dolphin\Dolphin.exe` with the argument `/e`
to start games. Lastly, it tells grundy that it should look for files ending
in `.gcm` when searching for games in a game collection that uses the
`dolphin` launcher. 

#### 5. Tell grundy where your game collections live
Once you have setup a game collection and a launcher, you will need to tell
grundy which launcher your collection uses.

You will want to copy the path to the collection. On Windows, you can do this
by opening the game collection in the Windows File Explorer and clicking on the
location bar. This will reveal the path, which will look something like:
`C:\Users\Me\Documents\My Games\gamecube-games`.

Now open up `app.grundy.ini` in a text editor (like Notepad). You will see
a line starting with `[game_collections]`. On the next line, paste your game
collection path and surround it with single quotes. **It is very important that
you put single quotes around the collection path**. This should be followed by
an equals character (`=`) and the name of the launcher to use for
the collection.

For example, if your `gamecube-games` collection lived in
`C:\Users\Me\Documents\My Games`, this file should now look
like this:
```ini
[settings]

[game_collections]
'C:\Users\Me\Documents\My Games\gamecube-games' = dolphin
```

#### 6. Reload Steam
Unfortunately, Steam needs to be restarted to learn about new shortcuts.
Once you have restarted Steam, you will see your new shind shortcuts.

## How do I build this?
Please refer to the [building documentation](docs/building).

## Are there advanced configuration options?
Please refer to the [configuration documentation](docs/configuration).
