#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>

int main() {
	char buf[64];
	gets(buf);
	system("id");
	popen("ls", "r");
	execve("/bin/sh", NULL, NULL);
	strcpy(buf, "x");
	strcat(buf, "y");
	memcpy(buf, "z", 1);
	socket(AF_INET, SOCK_STREAM, 0);
	/* bind all */
	const char *addr = "0.0.0.0";
	(void)addr;
	return 0;
}
