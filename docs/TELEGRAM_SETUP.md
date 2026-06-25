# Connecting Telegram to Postal

Unlike X, Instagram, Facebook, and TikTok (which use OAuth), Telegram connects
with **your own bot**. You create a bot, add it to the channel or group you want
to post to, and give Postal two values: the **bot token** and the **chat id**.
This is a one-time, ~3-minute setup.

The same steps are shown inline in the Postal **Channels → Telegram → Connect**
form (web and mobile).

---

## 1. Create a bot and get its token

1. Open Telegram and search for **@BotFather** (the official bot, blue check).
2. Start a chat and send **`/newbot`**.
3. BotFather asks for:
   - a **name** (display name, anything, e.g. "My Brand Poster"), and
   - a **username** that must end in `bot` (e.g. `mybrand_poster_bot`).
4. BotFather replies with your **bot token**, which looks like:

   ```
   123456789:AAE_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
   ```

   Keep this secret — it's the password to your bot. (You can revoke/regenerate
   it any time from BotFather with `/token`.)

## 2. Add the bot to your channel or group as an admin

Postal posts *through* your bot, so the bot must be a member with permission to
post.

- **Channel:** open the channel → **Administrators** → **Add Admin** → search for
  your bot's username → enable **Post Messages** → save.
- **Group:** open the group → **Add Members** → add your bot, then promote it to
  **Administrator** (enable at least **Send Messages**).

## 3. Find the chat id

The chat id tells the bot *where* to post. You have two cases:

- **Public channel/group (has a @username):** just use `@yourchannel` —
  that's the chat id. Simplest option.
- **Private channel/group (no public username):** you need the numeric id, which
  looks like `-1001234567890`. Easiest ways to get it:
  1. In Telegram, **forward any message** from your channel/group to
     **@userinfobot** (or **@getidsbot**). It replies with the chat's numeric id.
  2. Or, after the bot is an admin, post any message in the chat and open
     `https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates` in a browser — find
     `"chat":{"id":-100...}` in the JSON.

  > Numeric ids for channels/supergroups start with `-100`. Include the minus
  > sign and the full number.

## 4. Connect in Postal

1. Go to **Channels** in Postal and find **Telegram** under "Connect a platform".
2. Click **Connect**, then paste:
   - **Bot token** — from step 1 (`123456789:AAE_...`).
   - **Chat ID or @channel** — from step 3 (`@yourchannel` or `-1001234567890`).
3. Click **Connect**. Postal validates the token and that the bot can see the
   chat, then the channel appears under your connected accounts.

You can now compose and schedule posts to Telegram like any other channel. Text,
links, a single image, or a single video are supported. Telegram publishing is
**free** (no wallet credits).

---

## Troubleshooting

- **"could not connect: ... Unauthorized"** — the bot token is wrong or was
  revoked. Re-copy it from BotFather (`/token`).
- **"could not connect: ... chat not found"** — the chat id is wrong, or the bot
  hasn't been added to the chat yet. Double-check step 2 and 3.
- **Posts fail with a permissions error** — the bot is in the chat but isn't an
  admin / lacks "Post Messages". Re-check step 2.
- **Private channel id** — make sure you included the leading `-100`.
