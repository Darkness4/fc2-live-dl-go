CC = gcc
CFLAGS = -Wall -Wextra -I. -g
LDFLAGS = -lavformat -lavcodec -lavutil

SRCS = remux.c remux_test.c
OBJS = $(SRCS:.c=.o)

TARGET = remux_test

.PHONY: all clean

all: $(TARGET)

$(TARGET): $(OBJS)
	$(CC) $(LDFLAGS) -o $@ $^

%.o: %.c
	$(CC) $(CFLAGS) -c -o $@ $<

clean:
	rm -f $(TARGET) $(OBJS)

