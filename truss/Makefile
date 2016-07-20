all: truss

# -ignore swp ignores vim swap files
gobindata:
	cd data && \
	go-bindata -pkg data -o data.go -ignore swp service/...

# Install truss and all dependencies 
truss: gobindata
	go install github.com/TuneLab/gob/...

# Remove the truss binary
clean:
	rm $GOBIN/truss

# Run truss on all proto files in all dirs in ./tests/
test:
	./tests/test.sh ./tests/

# Remove the dir named `service/` in all dirs in ./tests/
testclean:
	./tests/cleantest.sh ./tests/


