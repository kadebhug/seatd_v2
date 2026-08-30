# Display and Device Management Runbook

## Pairing

1. In the owner workspace, open **Devices** and generate a pairing code for the target location.
2. On the display, enter the code before it expires.
3. The API consumes the code once, creates a trusted display device, and returns a revocable device credential.
4. The display stores the credential in platform secure storage and uses it for heartbeat and display snapshot requests.

Pairing codes are short lived, single use, scoped to one organisation/location, and stored by lookup prefix plus SHA-256 hash. Raw device credentials are returned only once at pairing time.

## Health

Displays send a heartbeat with app version and capabilities. Owner fleet health treats a trusted device as online when `lastHeartbeatAt` is within three configured heartbeat intervals, with a minimum threshold of three minutes.

Revoking a device marks the device revoked and revokes active credentials. A revoked credential can no longer heartbeat or load the display snapshot.

## Offline Display Behaviour

The display caches the last successful location snapshot locally. When network calls fail it keeps rendering that snapshot, marks the display offline, and continues retrying. It does not hide the stale state indicator.

## Android Deployment Baseline

Initial managed profile:

- one approved Android tablet/display SKU per rollout;
- landscape orientation for venue displays;
- launch-on-boot and lock-task/kiosk mode through Android Enterprise or the venue MDM;
- screen wake policy managed by device owner/MDM where available;
- APK distribution through the managed store or controlled sideload channel.

The Flutter app reports `displayAppVersion` with each heartbeat. Minimum supported versions should be enforced server-side only when compatibility requires it; until then, owners should stage APK releases by venue/location and roll back by redeploying the previous approved APK through the same channel.
