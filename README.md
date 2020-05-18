# hlc - Happy Learning Calendar
Record your happy learning life for you

## Usage
```
NAME:
   hlc - record happy learning life

USAGE:
   hlc [global options] command [command options] [arguments...]

VERSION:
   1.0.0

COMMANDS:
   init     init database
   list, l  show tasks today
   add, a   add a task you learn today
   done, d  done a task
   cal, c   show calendar
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

add a task you learn today
```
$ hlc add
```

list tasks
```
$ hlc list
```

done the task
```
$ hlc done [number]
```

show calendar
```
$ hlc list
```

## Installation
```
$ go get github.com/anyy/hlc
$ hlc init
```
