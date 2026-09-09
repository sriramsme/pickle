# Notifications

Pickle can send standard Web Push notifications to devices that opt in.

Open **Settings** and choose **Enable** under Notifications. Use **Send test**
to verify delivery before relying on notifications from future agent
integrations.

Notifications require HTTPS or localhost. On iPhone and iPad, Pickle must be
opened from an installed Home Screen icon before it can ask for notification
permission.

Pickle stores its VAPID keys and device subscriptions at:

```text
~/.config/pickle/notifications.json
```

The file is created with permissions `0600`. Removing it creates a new identity
for the Pickle server and invalidates existing device subscriptions. Turn
notifications off and on again on each device after removing it.
