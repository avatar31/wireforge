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
#define MY_SOCK "/tmp/c_listen.sock"
#define TARGET_SOCK "/tmp/go_listen.sock"

int outbound_sock = -1;
pthread_mutex_t out_mutex = PTHREAD_MUTEX_INITIALIZER;

// Thread-safe wrapper to handle JSON encoding and writes
void send_json_message(const char* sender, const char* content) {
    char payload[BUFFER_SIZE];
    char time_str[64];

    time_t now = time(NULL);
    struct tm *t = localtime(&now);
    strftime(time_str, sizeof(time_str), "%H:%M:%S", t);

    pthread_mutex_lock(&out_mutex);
    if (outbound_sock != -1) {
        // Enforce exact Line-Delimited JSON with explicit trailing '\n'
        snprintf(payload, sizeof(payload), 
                 "{\"sender\":\"%.127s\",\"content\":\"%.1023s\",\"timestamp\":\"%.63s\"}\n", 
                 sender, content, time_str);
                 
        if (send(outbound_sock, payload, strlen(payload), MSG_NOSIGNAL) < 0) {
            close(outbound_sock);
            outbound_sock = -1;
            if (strcmp(sender, "system") != 0) {
                printf("\r\33[2K[System] Outbound pipe broken. Reconnecting in background...\n> ");
                fflush(stdout);
            }
        }
    } else if (strcmp(sender, "system") != 0) {
        printf("[System] Cannot send. Go listener is currently offline.\n");
    }
    pthread_mutex_unlock(&out_mutex);
}

// Inbound Reader Thread (Parses JSON streaming from Go)
void* inbound_reader_thread(void* arg) {
    int server_sock = socket(AF_UNIX, SOCK_STREAM, 0);
    unlink(MY_SOCK);

    struct sockaddr_un addr;
    memset(&addr, 0, sizeof(addr));
    addr.sun_family = AF_UNIX;
    strncpy(addr.sun_path, MY_SOCK, sizeof(addr.sun_path) - 1);

    if (bind(server_sock, (struct sockaddr*)&addr, sizeof(addr)) < 0) {
        perror("C Bind failed");
        exit(1);
    }
    listen(server_sock, 1);

    char buffer[BUFFER_SIZE];
    while (1) {
        int client_sock = accept(server_sock, NULL, NULL);
        if (client_sock < 0) continue;

        printf("\r\33[2K[System] Go Peer connected to our listener!\n> ");
        fflush(stdout);

        // Configure a subtle read timeout so if Go freezes/crashes, read() unblocks
        struct timeval tv;
        tv.tv_sec = 3; 
        tv.tv_usec = 0;
        setsockopt(client_sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof(tv));

        while (1) {
            memset(buffer, 0, BUFFER_SIZE);
            ssize_t bytes_read = read(client_sock, buffer, BUFFER_SIZE - 1);
            if (bytes_read <= 0) break; // Lost connection or timeout passed without heartbeats

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

                // Filter out background heartbeats from spamming the UI
                if (strcmp(sender, "system") == 0 && strcmp(content, "ping") == 0) {
                    continue;
                }

                printf("\r\33[2K[%s] %s: %s\n> ", timestamp, sender, content);
                fflush(stdout);
            }
        }
        close(client_sock);
        printf("\r\33[2K[System] Go Peer disconnected from our listener.\n> ");
        fflush(stdout);
    }
    close(server_sock);
    return NULL;
}

// Outbound Dialer Thread
void* outbound_dialer_thread(void* arg) {
    struct sockaddr_un addr;
    memset(&addr, 0, sizeof(addr));
    addr.sun_family = AF_UNIX;
    strncpy(addr.sun_path, TARGET_SOCK, sizeof(addr.sun_path) - 1);

    while (1) {
        pthread_mutex_lock(&out_mutex);
        int need_connect = (outbound_sock == -1);
        pthread_mutex_unlock(&out_mutex);

        if (need_connect) {
            int sock = socket(AF_UNIX, SOCK_STREAM, 0);
            if (connect(sock, (struct sockaddr*)&addr, sizeof(addr)) == 0) {
                pthread_mutex_lock(&out_mutex);
                outbound_sock = sock;
                printf("\r\33[2K[System] Successfully connected to Go's listener!\n> ");
                fflush(stdout);
                pthread_mutex_unlock(&out_mutex);
            } else {
                close(sock);
            }
        }
        sleep(2);
    }
    return NULL;
}

// Heartbeat Daemon Thread
void* heartbeat_daemon_thread(void* arg) {
    while (1) {
        sleep(1);
        send_json_message("system", "ping");
    }
    return NULL;
}

// gcc main.c -o chatapp -pthread
int main() {
    printf("[System] Starting Resilient C Peer with JSON & Heartbeats...\n");

    pthread_t in_tid, out_tid, hb_tid;
    pthread_create(&in_tid, NULL, inbound_reader_thread, NULL);
    pthread_create(&out_tid, NULL, outbound_dialer_thread, NULL);
    pthread_create(&hb_tid, NULL, heartbeat_daemon_thread, NULL);
    pthread_detach(in_tid);
    pthread_detach(out_tid);
    pthread_detach(hb_tid);

    char input[1024];

    while (1) {
        printf("> ");
        fflush(stdout);
        if (!fgets(input, sizeof(input), stdin)) break;
        input[strcspn(input, "\n")] = 0;
        if (strlen(input) == 0) continue;

        if (strcmp(input, "exit") == 0) break;

        send_json_message("C-Peer", input);
    }

    unlink(MY_SOCK);
    return 0;
}
