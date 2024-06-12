CFLAGS=-Wall -Werror $(shell pkg-config --cflags libavformat libavcodec libavutil)
LDFLAGS=$(shell pkg-config --libs libavformat libavcodec libavutil)

VALGRIND_FLAGS=--leak-check=full --show-leak-kinds=all --track-origins=yes --error-exitcode=1

concat_valgrind_test.out: concat_valgrind_test.o concat.o
	$(CC) $(CFLAGS) -o $@ $^ $(LDFLAGS)

concat_valgrind_test.o: concat_valgrind_test.c
	$(CC) $(CFLAGS) -c -o $@ $^

concat.o: concat.c
	$(CC) $(CFLAGS) -c -o $@ $^

.PHONY: clean
clean:
	rm -f concat_valgrind_test.out concat_valgrind_test.o concat.o

.PHONY: valgrind
valgrind: concat_valgrind_test.out
	valgrind $(VALGRIND_FLAGS) ./concat_valgrind_test.out
