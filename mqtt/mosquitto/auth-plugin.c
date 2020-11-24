#include <string.h>
#include <stdlib.h>
#include <stdio.h>
#include <errno.h>

#include <mosquitto.h>
#include <mosquitto_plugin.h>

#if MOSQ_AUTH_PLUGIN_VERSION >= 3
#include <mosquitto_broker.h>
#endif

#include "go-auth.h"

int mosquitto_auth_plugin_version(void) {
#ifdef MOSQ_AUTH_PLUGIN_VERSION
  #if MOSQ_AUTH_PLUGIN_VERSION == 5
    return 4; // this is v2.0, use the backwards compatibility
  #else
    return MOSQ_AUTH_PLUGIN_VERSION;
  #endif
#else
    return 4;
#endif
}

int mosquitto_auth_plugin_init(void **user_data, struct mosquitto_opt *opts, int opt_count) {
    GoInt32 opts_count = opt_count;
    GoString keys[opt_count];
    GoString values[opt_count];
    int i;
    struct mosquitto_opt *o;

    for (i = 0, o = opts; i < opt_count; i++, o++) {
        GoString opt_key = {o->key, strlen(o->key)};
        GoString opt_value = {o->value, strlen(o->value)};
        keys[i] = opt_key;
        values[i] = opt_value;
    }

    GoSlice keysSlice = {keys, opt_count, opt_count};
    GoSlice valuesSlice = {values, opt_count, opt_count};

    if (AuthPluginInit(keysSlice, valuesSlice, opts_count)) {
        return MOSQ_ERR_SUCCESS;
    } else {
        return MOSQ_ERR_AUTH;
    }
}

int mosquitto_auth_plugin_cleanup(void *user_data, struct mosquitto_opt *opts, int opt_count) {
    AuthPluginCleanup();
    return MOSQ_ERR_SUCCESS;
}

int mosquitto_auth_security_init(void *user_data, struct mosquitto_opt *opts, int opt_count, bool reload) {
    return MOSQ_ERR_SUCCESS;
}

int mosquitto_auth_security_cleanup(void *user_data, struct mosquitto_opt *opts, int opt_count, bool reload) {
    return MOSQ_ERR_SUCCESS;
}

#if MOSQ_AUTH_PLUGIN_VERSION >= 4
int mosquitto_auth_unpwd_check(void *user_data, struct mosquitto *client, const char *username, const char *password)
#elif MOSQ_AUTH_PLUGIN_VERSION >= 3
int mosquitto_auth_unpwd_check(void *user_data, const struct mosquitto* client, const char *username, const char *password)
#else
int mosquitto_auth_unpwd_check(void *user_data, const char *usernamae, const char* password)
#endif
{
#if MOSQ_AUTH_PLUGIN_VERSION >= 3
    const char *clientid = mosquitto_client_id(client);
#else
    const char *clientid = "";
#endif

    if (username == NULL)
        username = "";
    if (password == NULL)
        password = "";

    GoString go_username = {username, strlen(username)};
    GoString go_password = {password, strlen(password)};
    GoString go_clientid = {clientid, strlen(clientid)};

    if (AuthUnpwdCheck(go_username, go_password, go_clientid)) {
        return MOSQ_ERR_SUCCESS;
    }

    return MOSQ_ERR_AUTH;
}

#if MOSQ_AUTH_PLUGIN_VERSION >= 4
int mosquitto_auth_acl_check(void *user_data, int access, struct mosquitto *client, const struct mosquitto_acl_msg *msg)
#elif MOSQ_AUTH_PLUGIN_VERSION >= 3
int mosquitto_auth_acl_check(void *user_data, int access, const struct mosquitto *client, const struct mosquitto_acl_msg *msg)
#else
int mosquitto_auth_acl_check(void *user_data, const char *clientid, const char *username, const char *topic, int access)
#endif
{
#if MOSQ_AUTH_PLUGIN_VERSION >= 3
    const char *clientid = mosquitto_client_id(client);
    const char *username = mosquitto_client_username(client);
    const char *topic = msg->topic;
#endif

    if (topic == NULL || access < 1) {
        printf("error: received null topic, or access is equal or less than 0 for acl check\n");
        fflush(stdout);
        return MOSQ_ERR_ACL_DENIED;
    }

    if (clientid == NULL)
        clientid = "";
    if (username == NULL)
        username = "";

    GoString go_clientid = {clientid, strlen(clientid)};
    GoString go_username = {username, strlen(username)};
    GoString go_topic = {topic, strlen(topic)};
    GoInt32 go_access = access;

    if (AuthAclCheck(go_clientid, go_username, go_topic, go_access)) {
        return MOSQ_ERR_SUCCESS;
    }

    return MOSQ_ERR_ACL_DENIED;
}

#if MOSQ_AUTH_PLUGIN_VERSION >= 4
int mosquitto_auth_psk_key_get(void *user_data, struct mosquitto *client, const char *hint, const char *identity, char *key, int max_key_len)
#elif MOSQ_AUTH_PLUGIN_VERSION >= 3
int mosquitto_auth_psk_key_get(void *user_data, const struct mosquitto *client, const char *hint, const char *identity, char *key, int max_key_len)
#else
int mosquitto_auth_psk_key_get(void *user_data, const char *hint, const char *identity, char *key, int max_key_len)
#endif
{
  return MOSQ_ERR_AUTH;
}
