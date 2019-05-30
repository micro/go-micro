# Discord input for micro-bot
[Discord](https://discordapp.com) support for micro bot based on [discordgo](github.com/bwmarrin/discordgo).

This was originally written by Aleksandr Tihomirov (@zet4) and can be found at https://github.com/zet4/micro-misc/.

## Options
### discord_token

You have to supply an application token via `--discord_token`.

Head over to Discord's [developer introduction](https://discordapp.com/developers/docs/intro)
to learn how to create applications and how the API works.

### discord_prefix

Set a command prefix with `--discord_prefix`. The default prefix is `Micro `.
You can mention the bot or use the prefix to run a command.

### discord_whitelist

Pass a list of comma-separated user IDs with `--discord_whitelist`. Only allow
these users to use the bot.
