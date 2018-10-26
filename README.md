# go-prochost
Process host mainly useful for gameservers. Creates a socket for communicating with an application running in the background.

It can buffer a given amount of output lines from the hosted process using the command-line parameter `-b`. The buffered lines will be sent out upon connecting to the socket created by prochost.

**Only tested to work on linux**

## Usage
```
Usage: prochost [-b bufsize] [-l listen] -f file [-- [command-line arguments to the executed file]]
  -b int
        Amount of lines to store in buffer
  -l string
        Listen path/address
  -f string
        File to execute
```

Command-line arguments can be passed to the executed file by adding them at the end of the parameters to prochost, preceeded by two dashes (`--`).

### Example usage
```
prochost -b 1000 -l /path/to/minecraft.sock -f /usr/bin/java -- -Xms1024M -Xmx2048M -jar /path/to/minecraft_server.jar nogui
```