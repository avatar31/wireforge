#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <pthread.h>
#include <stdint.h>
#include <time.h>

#define BUFFER_SIZE 2048
#define SOCKET_PATH "/tmp/chat.sock"

int server_sock = -1;
pthread_mutex_t sock_mutex = PTHREAD_MUTEX_INITIALIZER;

// Thread-safe wrapper to handle serialization and socket writing
void send_json_payload(const char* sender, const char* content) {
    char payload[BUFFER_SIZE];
    char time_str[64];

    time_t now = time(NULL);
    struct tm *t = localtime(&now);
    strftime(time_str, sizeof(time_str), "%H:%M:%S", t);

    pthread_mutex_lock(&sock_mutex);
    if (server_sock != -1) {
        // Build single-line explicit JSON string ending with a newline
        snprintf(payload, sizeof(payload), 
                 "{\"sender\":\"%.127s\",\"content\":\"%.1023s\",\"timestamp\":\"%.63s\"}\n", 
                 sender, content, time_str);
                 
        if (send(server_sock, payload, strlen(payload), MSG_NOSIGNAL) < 0) {
            close(server_sock);
            server_sock = -1;
            if (strcmp(sender, "system") != 0) {
                printf("\r\33[2K[Client] Lost connection to Server. Reconnecting...\n> ");
                fflush(stdout);
            }
        }
    } else if (strcmp(sender, "system") != 0) {
        printf("[Client] Cannot send. Server is currently offline.\n");
    }
    pthread_mutex_unlock(&sock_mutex);
}

// Inbound Reader Thread: Parses JSON streamed down from the Go server
void* server_reader_thread(void* arg) {
    int sock = (int)(intptr_t)arg;
    char buffer[BUFFER_SIZE];

    // Configure a 3-second receive timeout to clear zombie locks if the server hard-crashes
    struct timeval tv;
    tv.tv_sec = 3;
    tv.tv_usec = 0;
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof(tv));

    while (1) {
        memset(buffer, 0, BUFFER_SIZE);
        ssize_t bytes_read = read(sock, buffer, BUFFER_SIZE - 1);
        if (bytes_read <= 0) break; // Disconnect or timeout occurred

        char sender[128] = {0};
        char content[1024] = {0};
        char timestamp[64] = {0};

        char* s_ptr = strstr(buffer, "\"sender\":\"");
        char* c_ptr = strstr(buffer, "\"content\":\"");
        char* t_ptr = strstr(buffer, "\"timestamp\":\"");

        if (s_ptr && c_ptr && t_ptr) {
            sscanf(s_ptr, "\"sender\":\"%127[^\"]\"", sender);
            sscanf(c_ptr, "\"content\":\"%1023[^\"]\"", content);
            sscanf(t_ptr, "\"timestamp\":\"%63[^\"]\"", timestamp);

            // Filter out system heartbeats from printing
            if (strcmp(sender, "system") == 0 && strcmp(content, "ping") == 0) {
                continue;
            }

            printf("\r\33[2K[%s] %s: %s\n> ", timestamp, sender, content);
            fflush(stdout);
        }
    }

    pthread_mutex_lock(&sock_mutex);
    if (server_sock == sock) {
        close(server_sock);
        server_sock = -1;
        printf("\r\33[2K[Client] Server disconnected. Reconnecting in background...\n> ");
        fflush(stdout);
    }
    pthread_mutex_unlock(&sock_mutex);

    return NULL;
}

// Active Reconnection Dialer Thread
void* active_reconnect_thread(void* arg) {
    struct sockaddr_un addr;
    memset(&addr, 0, sizeof(addr));
    addr.sun_family = AF_UNIX;
    strncpy(addr.sun_path, SOCKET_PATH, sizeof(addr.sun_path) - 1);

    while (1) {
        pthread_mutex_lock(&sock_mutex);
        int need_connect = (server_sock == -1);
        pthread_mutex_unlock(&sock_mutex);

        if (need_connect) {
            int sock = socket(AF_UNIX, SOCK_STREAM, 0);
            if (connect(sock, (struct sockaddr*)&addr, sizeof(addr)) == 0) {
                pthread_mutex_lock(&sock_mutex);
                server_sock = sock;
                printf("\r\33[2K[Client] Successfully connected to Go Server!\n> ");
                fflush(stdout);

                pthread_t tid;
                pthread_create(&tid, NULL, server_reader_thread, (void*)(intptr_t)sock);
                pthread_detach(tid);
                pthread_mutex_unlock(&sock_mutex);
            } else {
                close(sock);
            }
        }
        sleep(2); // Retry polling cadence
    }
    return NULL;
}

// Heartbeat Generation Thread
void* client_heartbeat_thread(void* arg) {
    while (1) {
        sleep(1);
        send_json_payload("system", "ping");
    }
    return NULL;
}

// gcc main.c -o chatapp -pthread
int main() {
    printf("[Client] Initializing C Client...\n");

    pthread_t dial_tid, hb_tid;
    pthread_create(&dial_tid, NULL, active_reconnect_thread, NULL);
    pthread_create(&hb_tid, NULL, client_heartbeat_thread, NULL);
    pthread_detach(dial_tid);
    pthread_detach(hb_tid);

    char input[1024];

    while (1) {
        printf("> ");
        fflush(stdout);
        if (!fgets(input, sizeof(input), stdin)) break;
        input[strcspn(input, "\n")] = 0;
        if (strlen(input) == 0) continue;

        if (strcmp(input, "exit") == 0) {
            printf("[Client] Exiting...\n");
            break;
        }

        send_json_payload("Client (C)", input);
    }

    pthread_mutex_lock(&sock_mutex);
    if (server_sock != -1) close(server_sock);
    pthread_mutex_unlock(&sock_mutex);

    return 0;
}
