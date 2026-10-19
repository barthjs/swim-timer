---
title: Webhooks
description: Events, message format, delivery, and signature of the webhooks.
sidebar:
  order: 5
---

[Webhooks](../guides/webhooks.md) in the guides describes how to set up a webhook.

## Secret and address

- A secret has the format of the Standard Webhooks specification, `whsec_` followed by 24 to 64 bytes in base64.
- Several webhooks may share a secret, and a changed secret takes effect with the next message.
- The signature lets the receiver check every message with http or https, while https also keeps others
  from reading them.
- The certificate of an https receiver is verified unless the webhook accepts any certificate.
- A URL with an address in a private or local network is rejected on save, since a webhook could otherwise reach
  other services of that network. A host name is resolved on every delivery, and a delivery to such an address fails.
- The allowed networks, `WEBHOOK_ALLOWED_NETWORKS`, are exempt and take effect after a restart.

## Events

| Event           | Sent                                           | Data                                         |
| --------------- | ---------------------------------------------- | -------------------------------------------- |
| `heat.updated`  | after every change of the live heat            | the live heat as the overlays receive it     |
| `meet.reloaded` | after every successful reload of the meet data | the whole meet as the meet overview shows it |

A message is a JSON object:

```json
{
  "id": "msg_2B7VQ4XKFJ3M6N5P8R9S0T1UVW",
  "type": "heat.updated",
  "createdAt": "2026-10-19T10:15:30.123Z",
  "data": {}
}
```

`createdAt` is the time of Swim-Timer when it created the message. A receiver can use it to calculate
the running time from the start time of the live heat even if its own clock differs. Heats are counted from 1.

## Delivery

Every webhook has its own queue and sends one message at a time, so a slow receiver delays neither the overlays
nor other webhooks. A new message replaces a waiting message of the same event, so a slow receiver gets fewer
intermediate states but always the current one. The events take turns, so a failing message of one event
does not hold back the other.

The receiver confirms a message with a status from 200 to 299. Other answers are handled as follows:

| Answer                                                     | Handling                                                                                                                                                              |
| ---------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 410                                                        | The webhook is turned off, since the receiver no longer exists.                                                                                                       |
| 429 or 503 with `Retry-After`                              | The message is sent again after the requested wait, at most five minutes.                                                                                             |
| 408, 429, 5xx, no answer within ten seconds, network error | The message is sent again after a wait that starts at one second and doubles up to one minute. A newer message of the same event replaces it before the next attempt. |
| any other status, including a redirect                     | The message is dropped. Redirects are not followed.                                                                                                                   |

A webhook is never turned off for failed deliveries alone. A message can arrive twice, for example if the answer
of the receiver got lost, with the same `id` both times. After a restart, the deliveries start with the next change.

## Verifying a message

Every request carries three headers of the Standard Webhooks specification:

| Header              | Content                                                                                  |
| ------------------- | ---------------------------------------------------------------------------------------- |
| `webhook-id`        | the `id` of the message                                                                  |
| `webhook-timestamp` | the time of the request in seconds since 1970                                            |
| `webhook-signature` | `v1,` followed by the HMAC-SHA256 of `<webhook-id>.<webhook-timestamp>.<body>` in base64 |

The key of the HMAC is the part of the secret after `whsec_`, decoded from base64. The libraries of the specification
verify a message in one call and reject a message whose timestamp differs from their clock by more than five minutes.
The clock of the computer running Swim-Timer therefore has to be correct.
