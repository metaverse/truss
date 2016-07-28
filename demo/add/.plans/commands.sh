
function vimp {
	rm -f /home/adamryman/projects/go/src/github.com/TuneLab/gob/demo/add/service.proto
	vim +'set ft=proto'\
		+"PlayMeOff /home/adamryman/projects/go/src/github.com/TuneLab/gob/demo/add/.plans/protofiles/0010_base.proto"\
	   	+'save /home/adamryman/projects/go/src/github.com/TuneLab/gob/demo/add/service.proto'
}
