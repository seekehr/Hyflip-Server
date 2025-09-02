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

As such, I can allow... (i forgot what i was typing)
# FLIPPER FLOW

First of all, make an API request to the BZ/AH API. This opens up a SSE stream between the server-client allowing quick updates
instead of a once-update (which would take minimum 8-7s prob even with good wifi).

Update once every 21s, allowing 1s for API to update.
### Bazaar Flipping

So, for bazaar flipping, we have a bunch of stats. But for now, it will basically scour the bazaar endpoint and look at all the items. 
And look for (e.g) `sell - buy > minProfitUserWants` or stuff like `volume > minVolumeUserWants`, and stuff like `excludeItemsUserWants()`. For that,
I'll need a config file that each user can create easily and can also share them. How should I achieve this?

So we access the API, which contains all items and will probably take up 100-200MB of memory but no big deal. So we check for
profit (initialised in a default config).

Now, another thing: `caching`. To allow many people to use the app, we need to cache as it's not worth it to make 100s of 
requests to the Hypixel API every 20 seconds (rate-limits don't exist on bazaar endpoint, but caching also allows us to not
make the user unnecessarily wait 4-5 seconds to start receiving updates). As such, we'll make a universal cache store because
why NOT 

# TODO


# DONE

1. Buffer Everything During Update

When you start the update, create a temporary slice (buffer).

Every item the API produces is sent to the live channel and appended to this buffer.

2. Users Joining Mid-Update

When a user connects mid-update:

First, they read all items currently in the buffer.

Then, they continue reading new items from the live channel as they arrive.

This ensures no user misses anything, even if they join 30 seconds into the 50-second update.

3. After Update Finishes

Save the final buffer as the snapshot.

Clear the live buffer (or keep it if you want to allow replay for very late joiners).

Mark the update as finished.

4. Key Points

Single update goroutine handles API fetch.

Buffer + live channel guarantees mid-update users get all data.

Snapshot guarantees new users after the update still have data