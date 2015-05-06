PROGRAM = ./bin/osx-trash
DEST    = /usr/local/bin

all: install

install:
	install -s $(PROGRAM) $(DEST)
