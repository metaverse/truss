all: truss

# -ignore swp ignores vim swap files
gobindata:
	cd data && \
	go-bindata -pkg data -o data.go -ignore swp service/...

truss: gobindata
	go install github.com/TuneLab/gob/truss/cmd/...

clean:
	rm -r service

test:
	truss addsvc.proto

