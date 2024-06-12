CFLAGS=-Wall -Werror $(shell pkg-config --cflags libavformat libavcodec libavutil)
LDFLAGS=$(shell pkg-config --libs libavformat libavcodec libavutil)

VALGRIND_FLAGS=--leak-check=full --show-leak-kinds=all --track-origins=yes --error-exitcode=1

probe_valgrind_test.out: probe_valgrind_test.o probe.o
	$(CC) $(CFLAGS) -o $@ $^ $(LDFLAGS)

probe_valgrind_test.o: probe_valgrind_test.c
	$(CC) $(CFLAGS) -c -o $@ $^

probe.o: probe.c
	$(CC) $(CFLAGS) -c -o $@ $^

.PHONY: clean
clean:
	rm -f probe_valgrind_test.out probe_valgrind_test.o probe.o

.PHONY: valgrind
valgrind: probe_valgrind_test.out
	valgrind $(VALGRIND_FLAGS) ./probe_valgrind_test.out probe
	valgrind $(VALGRIND_FLAGS) ./probe_valgrind_test.out contains_video_or_audio
	valgrind $(VALGRIND_FLAGS) ./probe_valgrind_test.out is_mpegts_or_aac
