#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <pthread.h>
#include <stdint.h>
#include <time.h>

#include "messages.h"

#define SOCKET_PATH "/tmp/chat.sock"

int server_sock = -1;
int joined = 0;
pthread_mutex_t sock_mutex = PTHREAD_MUTEX_INITIALIZER;

// Helper to reliably read explicit sizes from a streaming socket
static int read_all(int sock, uint8_t* dest, size_t len) {
    size_t total_read = 0;
    while (total_read < len) {
        ssize_t r = recv(sock, dest + total_read, len - total_read, 0);
        if (r <= 0) return -1; // Error or socket EOF dropped
        total_read += r;
    }
    return 0;
}

void send_message(uint8_t* buf, int len) {
    pthread_mutex_lock(&sock_mutex);
    if (server_sock != -1) {
        if (send(server_sock, buf, len, MSG_NOSIGNAL) < 0) {
            close(server_sock);
            server_sock = -1;
            joined = 0;
            printf("\r\33[2K User Disconnected\n> ");
            fflush(stdout);
        }
    }
    pthread_mutex_unlock(&sock_mutex);
}

// Fixed Reader Thread: Uses read_all to handle streaming socket boundaries safely
void* server_reader_thread(void* arg) {
    int sock = (int)(intptr_t)arg;
    struct timeval tv = { .tv_sec = 5, .tv_usec = 0 };
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof(tv));

    uint8_t frame[WIRE_FRAME_HEADER_SIZE];
    while (1) {
        if (read_all(sock, frame, WIRE_FRAME_HEADER_SIZE) != 0) break;

        uint16_t type_id = get_message_type(frame);
        uint16_t fixed_len = get_message_fixed_length(frame);

        // Allocate and read remaining structure frame data dynamically based on length
        uint8_t* fixed_buf = malloc(fixed_len);
        if (!fixed_buf) break;

        if (read_all(sock, fixed_buf, fixed_len) != 0) {
            free(fixed_buf);
            break;
        }

        if (type_id == USER_MESSAGE_TYPE_ID) {
            size_t dyn_total = calculate_usermessage_dynamic_payload_size(fixed_buf);

            size_t full_payload_len = fixed_len + dyn_total;
            uint8_t* full_payload = malloc(full_payload_len);
            if (!full_payload) {
                free(fixed_buf);
                break;
            }

            memcpy(full_payload, fixed_buf, fixed_len);
            free(fixed_buf);

            if (dyn_total > 0) {
                if (read_all(sock, full_payload + fixed_len, dyn_total) != 0) {
                    free(full_payload);
                    break;
                }
            }

            UserMessage_t msg = {0};
            if (usermessage_unmarshal(full_payload, full_payload_len, fixed_len, &msg) == 0) {
                printf("\r\33[2K[User] %s\n> ", msg.content ? msg.content : "");
                fflush(stdout);
                usermessage_free(&msg);
            }
            free(full_payload);

        } else if (type_id == HEARTBEAT_MESSAGE_TYPE_ID) {
            HeartbeatMessage_t hb = {0};
            heartbeatmessage_unmarshal(fixed_buf, fixed_len, fixed_len, &hb);
            free(fixed_buf);
            // Clean pass on heartbeat frame logic sync. Loop repeats.
        } else {
            free(fixed_buf);
            break; 
        }
    }

    pthread_mutex_lock(&sock_mutex);
    if (server_sock == sock) {
        close(server_sock);
        server_sock = -1;
        joined = 0;
        printf("\r\33[2K User left chat...\n> ");
        fflush(stdout);
    }
    pthread_mutex_unlock(&sock_mutex);
    return NULL;
}

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
            if (sock >= 0) {
                if (connect(sock, (struct sockaddr*)&addr, sizeof(addr)) == 0) {
                    pthread_mutex_lock(&sock_mutex);
                    server_sock = sock;
                    joined = 1;
                    printf("\rJoined to chat.\n> ");
                    fflush(stdout);

                    pthread_t tid;
                    pthread_create(&tid, NULL, server_reader_thread, (void*)(intptr_t)sock);
                    pthread_detach(tid);
                    pthread_mutex_unlock(&sock_mutex);
                } else {
                    close(sock);
                }
            }
        }
        sleep(2);
    }
    return NULL;
}

void* client_heartbeat_thread(void* arg) {
    while (1) {
        sleep(2);
        if (!joined) continue;

        uint8_t* buf = NULL;
        HeartbeatMessage_t hb = {0};
        heartbeatmessage_set_timestamp(&hb, (int64_t)time(NULL));
        
        int len = heartbeatmessage_marshal(&hb, &buf);
        if (len > 0 && buf) {
            send_message(buf, len);
        }
        free(buf);
    }
    return NULL;
}

int build_usermessage(uint8_t** out_buf, const char* message) {
    UserMessage_t msg = {0};
    usermessage_set_timestamp(&msg, (int64_t)time(NULL));
    usermessage_set_content(&msg, message);

    if (msg.content == NULL) {
        usermessage_free(&msg);
        return -1;
    }

    int total = usermessage_marshal(&msg, out_buf);
    usermessage_free(&msg);
    return total;
}

// gcc main.c messages.c -o chatapp -pthread
int main() {
    system("clear");
    printf("=========== Welcome ===========\n");

    pthread_t dial_tid, hb_tid;
    pthread_create(&dial_tid, NULL, active_reconnect_thread, NULL);
    pthread_create(&hb_tid, NULL, client_heartbeat_thread, NULL);
    pthread_detach(dial_tid);
    pthread_detach(hb_tid);

    printf("Type 'exit' to close the application.\n");

    char input[1024];
    while (1) {
        printf("> ");
        fflush(stdout);
        if (!fgets(input, sizeof(input), stdin)) break;
        input[strcspn(input, "\n")] = 0;
        if (strlen(input) == 0) continue;

        if (strcmp(input, "exit") == 0) {
            printf("Leaving chat...\n");
            break;
        }

        if (!joined) {
            printf("Wait for user to join before sending messages.\n");
            continue;
        }

        uint8_t* buf = NULL;
        int len = build_usermessage(&buf, input);
        if (len > 0 && buf) {
            send_message(buf, len);
        }
        free(buf);
    }

    pthread_mutex_lock(&sock_mutex);
    if (server_sock != -1) close(server_sock);
    pthread_mutex_unlock(&sock_mutex);
    return 0;
}
