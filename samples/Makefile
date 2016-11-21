all: Largo+from+-Concerto-No5_JS_Bach.pcm

Largo+from+-Concerto-No5_JS_Bach.pcm: Largo+from+-Concerto-No5_JS_Bach.mp3
	avconv -i Largo+from+-Concerto-No5_JS_Bach.mp3 -ac 1 -ar 11025 -f s16le Largo+from+-Concerto-No5_JS_Bach.pcm

play: 
	play -t raw -r 11025 -e signed -b 16 -c 1 Largo+from+-Concerto-No5_JS_Bach.pcm
