Bismillah

# USER FLOW

User purchases premium from me manually in a discord ticket using crypto. I simply add their name to "premium_players.txt".
Then, using a website/discord bot/mod or whatever, user clicks on Generate Key (and enter their username, or perhaps link with Discord first).
As such, `create_account` is called and their key is hashed (with username as salt) stored in the database alongside their UUID and username. 
`create_account` returns their key (unhashed, original one). Now the user can copy this key and /setkey in the mod OR if it's
website only then the key is simply stored as a cookie in their browser.

Then the `auth_middleware` comes into play for EVERY request. It takes the `token` (which is our key, unhashed as is saved in cookies/files),
and the `username`. It checks if the hashed token (with the username as salt) exists in the db, and if so it proceeds. Only issue is that
someone might just share their username and key and it would work. I'll fix this in the future though. Maybe use a discord oauth id as salt instead.

As such, I can allow the 
# FLIPPER FLOW

### Bazaar Flipping

So, for bazaar flipping, we have a bunch of stats. But for now, it will basically scour the bazaar endpoint and look at all the items. 
And look for (e.g) `sell - buy > minProfitUserWants` or stuff like `volume > minVolumeUserWants`, and stuff like `excludeItemsUserWants()`. For that,
I'll need a config file that each user can create easily and can also share them. How should I achieve this?

# TODO


# DONE