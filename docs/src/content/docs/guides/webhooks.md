---
title: Webhooks
description: Send the live heat and the meet data to a website.
sidebar:
  order: 7
---

A webhook sends the live heat or the meet data to a URL, for example to a website that shows the results
on the internet. Swim-Timer sends the data itself, so the receiver does not need to reach Swim-Timer
in the local network.

## Setting up a webhook

Webhooks are managed in the Webhooks tab of the configuration page and take effect right away. A webhook has:

| Field                  | Content                                                                                                   |
| ---------------------- | --------------------------------------------------------------------------------------------------------- |
| Name                   | A name to recognize it.                                                                                   |
| URL                    | The address the messages are sent to.                                                                     |
| Events                 | Live heat, sent after every change of the live heat, and meet data, sent after every reload.              |
| Secret                 | Signs every message, so the receiver can check that it comes from Swim-Timer. Generated on save if empty. |
| Accept any certificate | For a receiver with a self-signed certificate.                                                            |

- Show example shows the message a webhook would receive right now, without sending it.
- Send test delivers a test message and reports the answer of the receiver.
- A webhook may only send to a private or local network that is listed in the allowed networks of the tab.

## Delivery status

The list of webhooks shows whether the last delivery of each one succeeded, and Deliveries lists the last
deliveries with the answer of the receiver. A message that fails for a temporary reason is sent again.

[Webhooks](../reference/webhooks.md) in the reference describes the messages, the delivery,
and how a receiver verifies the signature.
