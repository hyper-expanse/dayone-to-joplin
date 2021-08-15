# dayone-to-joplin

> Import a Day One JSON export into the Joplin note-taking application with all journal entries and photos.

Tool to import journal entries, and most associated resources, from the [Day One](https://dayoneapp.com/) journal application into the [Joplin](https://joplinapp.org/) note-taking application.

## Table of Contents
<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [Contributing](#contributing)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Features

- [x] Support for [JSON export format](https://help.dayoneapp.com/en/articles/440668-exporting-entries) from Day One.
- [x] Import each journal entry along with their tags, location and date.
- [x] Import photo resource references.

## Installation

Git clone this repository onto a system with the [`Go`](https://golang.org/) programming language tool installed.

## Usage

Extract the JSON export from Day One into a folder.

Start Joplin note-taking application, navigate to the _Tools_ -> _Options_ -> _Web Clipper_ settings page and enable Web Clipper. Copy the token provided within the _Advance options_ box.

Query the Joplin API to find the ID of the notebook you want to import your journal entries into:

```bash
curl localhost:41184/folders?token=[TOKEN]
```

With the folder path to your exported journal, Joplin token and notebook ID, you can import your journal entries into Joplin by running the following command:

```bash
go run main.go --journalFolder [FULL PATH TO EXTRACTED JSON EXPORT FOLDER] --token [TOKEN] --notebook [NOTEBOOK ID]
```

## Contributing

Please read our [contributing guide](./contributing.md) to see how you may contribute to this project.
