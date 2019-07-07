# Marvin Dev

Development fork of Slack bot that uses the RTM API to interact with users using normal messages instead of slash commands.

## Code Structure

Typedefs are in interfaces.go and slack/.

The websocket connection code is in slack/rtm; the implementation of the Team type is the slack/controller package.

main() lives in cmd/slacktest. Some brief database infrastructure is in database/.

Most of the functionality lives in modules/. The `atcommand` module handles command parsing. The `factoid` module handles information storage/retrieval via factoids.

## License

Marvin is available of the terms of the GPLv3, or any later version at your discretion.
