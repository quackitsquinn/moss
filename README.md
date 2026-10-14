# Moss: A tool for randomly sampling from a large collection of files

This project started out of a problem I had:

I for a while have been using Apollo Guidance Computer (AGC) documents and blueprints as backgrounds for my desktop. I thought it was a fun idea, and I ended up writing a tiny tool to just invert all the images at once, since I prefer dark mode. However. I'm an all-or-nothing kind of person, and I wanted *all* the images.

## Usage

The server will need a config file to run. You can generate a default config with `mossctl server defconfig -c <config>`, and then edit it to your liking.

`mossctl gen-db --db <db> --source <source>`: Will generate a database of all the files in the source directory, and store it in the db directory.
`mossctl server defconfig -c <config>`: Will write the default config to the specified file.
`mossctl server start -c <config>`: Will start the server with the specified config.

This will host a HTTP server that, on any request, will return a random file from the database. With the `%n` query parameter, you can specify how many files to return. The server will return a zip file containing the requested number of files.

## How it works

The core of Moss is a simple recursive folder structure. Each folder contains up to 16 folders and 16 files, both named with a single character, 0-9 and a-f. Folders can contain other folders. Each file is a link to some outside source file. This recursive structure is filled top to bottom, 0 - f, with each child layer only being created when the parent layer is full. Because of this, it requires $O(log16(n)) disk seeks to find a random file, and the files are stored in a way that is easy to transport. The server will serve a random file from the database on each request, and can be used as a background source for your desktop. (or for randomly sampling from a large collection of files in general, though right now it's only *theoretically* uniformly random)

## Background

At first I just grabbed around 5~ collections from the [Internet Archive](https://archive.org/details/virtualagcproject), each one containing ~1k-2k and wrote a quick tool to invert them all, and tossed them in a folder. I set my desktop background to cycle through them, and.. well. problems pretty quickly arose. Most WMs don't like cycling through 10k images. On macOS, it creates a giant cache of all the images, taking up over 100GB of space. On Linux? Well, my WM of choice, Plasma, just.. shows a black screen. Besides the problems in the WMs themselves, there's also the problem of just.. transporting 10k images around. I wrote a tool to download all the AGC images, and ended up with a massive 54K IMAGES. That's where I realized I needed a better solution. Over a few days, I came up with this:

1. A tool that is as simple as a large collection of files, that can be randomly sampled from, and can be used as a background source for your desktop.
   - Ideally it'd avoid the 50k image folder problem somehow
   - That's this project!
2. A fancier background app that's designed around composability and animation and filters, half to fix the problem of "how do I get my desktop to use this URL as a background source", and half because I kinda miss Wallpaper Engine from before I switched to Linux.
   - I haven't quite started this yet. Right now I know it's gonna be configured using a declarative toml file (though, I'm not entirely opposed to lua or python) that ideally has a super nice no-code GUI for it. Only wrinkle is getting the window to behave like a desktop background.. I'm sure I'll figure it out.

Since this project is now at a mostly finished state for now, I have a [public instance](https://apollo.quackthe.duckspin.net) of Moss that you can use to get random AGC images. I might add a few more features, but for now, it serves the purposes that I need.
