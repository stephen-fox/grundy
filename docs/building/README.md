# Build documentation
This document explains how to build the application. Please note that the build
requirements stack as you move through the document. For example, building an
installer requires that you have fulfilled the requirements for building
the application.

A Gradle project is included to automate the more complex work involved in
building the application. **Gradle is not required for simply running the
application**. Build artifacts produced by the Gradle project are saved to a
directory named `build` in the root of the project.

## Running the application

#### Requirements
- Golang 1.10 or greater

#### Running from the command line
You can run the application by executing:
```bash
go run cmd/grundy/main.go
```

#### Running from an IDE
Simply execute the `main()` function in `cmd/grundy/main.go` using your IDE.

## Building the application

#### Requirements
- Java (to run Gradle)

#### Building from the command line:

###### Linux application
```bash
./gradlew buildExeLinux
```

###### macOS application
Note: You can build just the executable, or a .app:
```bash
# Single binary executable.
./gradlew buildExeMacos

# .app Application
./gradlew buildApplicationMacos
```

###### Windows application
```bash
./gradlew buildExeWindows
```

#### Building from an IDE
If your IDE has a Gradle plugin, make sure to install it. This will allow you
to execute Gradle tasks from your IDE.

## Building an installer

#### Linux requirements
- A Linux system
- fpm

#### macOS requirements
- A macOS system

#### Windows requirements
Note: I recommend using chocolately to install dependencies to make your
life easier:

- A Windows system
- innosetup

#### Building from the command line

###### Linux installer
```bash
./gradlew buildInstallerLinux
```

###### macOS installer
```bash
./gradlew buildInstallerMacos
```

###### Windows installer
```bash
./gradlew buildInstallerWindows
```

#### Building from an IDE
If your IDE has a Gradle plugin, make sure to install it. This will allow you
to execute Gradle tasks from your IDE.
