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

#define BUFFER_SIZE 2048
#define MY_SOCK "/tmp/c_listen.sock"
#define TARGET_SOCK "/tmp/go_listen.sock"

typedef enum {
    MESSAGE_TYPE_USER_MESSAGE = 1,
    MESSAGE_TYPE_USER_LEFT = 2,
    MESSAGE_TYPE_USER_JOINED = 3,
    MESSAGE_TYPE_HEARTBEAT = 4
} msg_type_t;

int outbound_sock = -1;
int joined = 0;
char peer_name[256] = {0};
char myname[256] = {0};
pthread_mutex_t out_mutex = PTHREAD_MUTEX_INITIALIZER;

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

void send_message(msg_type_t msg_type, uint8_t* buf, int len) {
    pthread_mutex_lock(&out_mutex);
    if (outbound_sock != -1) {
        if (msg_type == MESSAGE_TYPE_USER_JOINED || joined) {
            if (send(outbound_sock, buf, len, MSG_NOSIGNAL) < 0) {
                close(outbound_sock);
                outbound_sock = -1;
                joined = 0;
                printf("\r\33[2K User Disconnected\n");
                fflush(stdout);
            }
        }
    }
    pthread_mutex_unlock(&out_mutex);
}

// Inbound Reader Thread (Parses JSON streaming from Go)
void* inbound_reader_thread(void* arg) {
    int in_sock = socket(AF_UNIX, SOCK_STREAM, 0);
    unlink(MY_SOCK);

    struct sockaddr_un addr;
    memset(&addr, 0, sizeof(addr));
    addr.sun_family = AF_UNIX;
    strncpy(addr.sun_path, MY_SOCK, sizeof(addr.sun_path) - 1);

    if (bind(in_sock, (struct sockaddr*)&addr, sizeof(addr)) < 0) {
        printf("Failed to open channel\n");
        close(in_sock);
        exit(1);
    }
    listen(in_sock, 1);

    char buffer[BUFFER_SIZE];
    while (1) {
        int client_sock = accept(in_sock, NULL, NULL);
        if (client_sock < 0) continue;

        fflush(stdout);

        // Configure a subtle read timeout so if Go freezes/crashes, read() unblocks
        struct timeval tv;
        tv.tv_sec = 3; 
        tv.tv_usec = 0;
        setsockopt(client_sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof(tv));

        uint8_t frame[WIRE_FRAME_HEADER_SIZE];
        while (1) {
            if (read_all(client_sock, frame, WIRE_FRAME_HEADER_SIZE) != 0) break;

            uint16_t type_id = get_message_type(frame);
            uint16_t fixed_len = get_message_fixed_length(frame);

            // Allocate and read remaining structure frame data dynamically based on length
            uint8_t* fixed_buf = malloc(fixed_len);
            if (!fixed_buf) break;

            if (read_all(client_sock, fixed_buf, fixed_len) != 0) {
                free(fixed_buf);
                break;
            }

            // Handle the message based on its type
            switch (type_id) {
                case MESSAGE_TYPE_USER_JOINED: {
                    size_t dyn_total = calculate_userjoinedmessage_dynamic_payload_size(fixed_buf);

                    size_t full_payload_len = fixed_len + dyn_total;
                    uint8_t* full_payload = malloc(full_payload_len);
                    if (!full_payload) {
                        free(fixed_buf);
                        break;
                    }

                    memcpy(full_payload, fixed_buf, fixed_len);
                    free(fixed_buf);

                    if (dyn_total > 0) {
                        if (read_all(client_sock, full_payload + fixed_len, dyn_total) != 0) {
                            free(full_payload);
                            break;
                        }
                    }

                    UserJoinedMessage_t msg = {0};
                    if (userjoinedmessage_unmarshal(full_payload, full_payload_len, fixed_len, &msg) == 0) {
                        joined = 1;
                        strncpy(peer_name, msg.username, sizeof(peer_name) - 1);
                        printf("\nUser %s joined chat...\n", msg.username);
                        userjoinedmessage_free(&msg);
                    }
                    free(full_payload);
                    break;
                }
                case MESSAGE_TYPE_USER_LEFT: {
                    size_t dyn_total = calculate_userleftmessage_dynamic_payload_size(fixed_buf);

                    size_t full_payload_len = fixed_len + dyn_total;
                    uint8_t* full_payload = malloc(full_payload_len);
                    if (!full_payload) {
                        free(fixed_buf);
                        break;
                    }

                    memcpy(full_payload, fixed_buf, fixed_len);
                    free(fixed_buf);

                    if (dyn_total > 0) {
                        if (read_all(client_sock, full_payload + fixed_len, dyn_total) != 0) {
                            free(full_payload);
                            break;
                        }
                    }

                    UserLeftMessage_t msg = {0};
                    if (userleftmessage_unmarshal(full_payload, full_payload_len, fixed_len, &msg) == 0) {
                        joined = 0;
                        peer_name[0] = '\0';
                        printf("\nUser %s left chat...\n", msg.username);
                        userleftmessage_free(&msg);
                    }
                    free(full_payload);
                    break;
                }
                case MESSAGE_TYPE_USER_MESSAGE: {
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
                        if (read_all(client_sock, full_payload + fixed_len, dyn_total) != 0) {
                            free(full_payload);
                            break;
                        }
                    }

                    UserMessage_t msg = {0};
                    if (usermessage_unmarshal(full_payload, full_payload_len, fixed_len, &msg) == 0) {
                        printf("\r\33[2K[%s] %s\n> ", peer_name, msg.content ? msg.content : "");
                        fflush(stdout);
                        usermessage_free(&msg);
                    }
                    free(full_payload);
                    break;
                }
                case MESSAGE_TYPE_HEARTBEAT: {
                    HeartbeatMessage_t hb = {0};
                    heartbeatmessage_unmarshal(fixed_buf, fixed_len, fixed_len, &hb);
                    free(fixed_buf);
                    // Clean pass on heartbeat frame logic sync. Loop repeats.
                    break;
                }
                default: {
                    free(fixed_buf);
                    break;
                }
            }
        }
        close(client_sock);
        printf("User %s left the chat...\n", peer_name);
        joined = 0;
        peer_name[0] = '\0';
        fflush(stdout);
    }
    close(in_sock);
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
                fflush(stdout);
                pthread_mutex_unlock(&out_mutex);

                uint8_t* buf = NULL;
                UserJoinedMessage_t join_msg = {0};
                userjoinedmessage_set_timestamp(&join_msg, (int64_t)time(NULL));
                userjoinedmessage_set_username(&join_msg, myname);

                int len = userjoinedmessage_marshal(&join_msg, &buf);
                if (len > 0 && buf) {
                    send_message(MESSAGE_TYPE_USER_JOINED, buf, len);
                }
                userjoinedmessage_free(&join_msg);
                free(buf);
            } else {
                close(sock);
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
            send_message(MESSAGE_TYPE_HEARTBEAT, buf, len);
        }
        heartbeatmessage_free(&hb);
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
    while (1) {
        printf("Enter your name: ");
        if (!fgets(myname, sizeof(myname), stdin)) break;
        myname[strcspn(myname, "\n")] = 0;

        if (strlen(myname) != 0) break;

        printf("Please enter your name to enter the chat.\n");
    }

    pthread_t in_tid, out_tid, hb_tid;
    pthread_create(&in_tid, NULL, inbound_reader_thread, NULL);
    pthread_create(&out_tid, NULL, outbound_dialer_thread, NULL);
    pthread_create(&hb_tid, NULL, client_heartbeat_thread, NULL);
    pthread_detach(in_tid);
    pthread_detach(out_tid);
    pthread_detach(hb_tid);

    printf("Type 'exit' to close the application.\n");

    char input[1024];
    while (1) {
        printf("%s> ", myname);
        fflush(stdout);
        if (!fgets(input, sizeof(input), stdin)) break;
        input[strcspn(input, "\n")] = 0;
        if (strlen(input) == 0) continue;

        if (strcmp(input, "exit") == 0) {
            printf("Leaving chat...\n");
            uint8_t* buf = NULL;
            UserLeftMessage_t left_msg = {0};
            userleftmessage_set_timestamp(&left_msg, (int64_t)time(NULL));
            userleftmessage_set_username(&left_msg, myname);
            
            int len = userleftmessage_marshal(&left_msg, &buf);
            if (len > 0 && buf) {
                send_message(MESSAGE_TYPE_USER_LEFT, buf, len);
            }
            userleftmessage_free(&left_msg);
            free(buf);
            break;
        }

        if (!joined) {
            printf("Wait for user to join before sending messages.\n");
            continue;
        }

        uint8_t* buf = NULL;
        int len = build_usermessage(&buf, input);
        if (len > 0 && buf) {
            send_message(MESSAGE_TYPE_USER_MESSAGE, buf, len);
        }
        free(buf);
    }

    unlink(MY_SOCK);
    return 0;
}
