all: Largo+from+-Concerto-No5_JS_Bach.pcm

Largo+from+-Concerto-No5_JS_Bach.pcm: Largo+from+-Concerto-No5_JS_Bach.mp3
	sox Largo+from+-Concerto-No5_JS_Bach.mp3 -t raw -e signed -r 11025 -b 16 -c 1 -L Largo+from+-Concerto-No5_JS_Bach.pcm

play: 
	play -t raw -r 11025 -e signed -b 16 -c 1 Largo+from+-Concerto-No5_JS_Bach.pcm
