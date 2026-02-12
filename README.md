# ShowMyCards

ShowMyCards is a self-hosted application for managing your Magic: The Gathering collection, using data from the most awesome [Scryfall][Scryfall].

You can learn more about ShowMyCards at [showmy.cards][ShowMyCards]

## Useful Links

- [Setup and Install](https://showmy.cards/install)
- [User Guide](https://showmy.cards/user)
- [Self Hosting](https://showmy.cards/selfhost)
- [API Reference](https://showmy.cards/api)

## Features

- Manage storage locations, such as binders and boxes
- Search and Bulk Data powered by Scryfall
- Master Lists - track collections seperately from inventory

ShowMyCards also features a rule evaluation engine - express your rules, and when a card is evaluated, it'll be sent to the right location.

More features are coming soon!

## Tech stack

- Backend: Go with Fiber, Gorm and Tygo
- Frontend: Typescript with SvelteKit and DaisyUI

## Why seperate the front and backend?

If you don't want to self host the UI, or want to build your own UI, you can; all that's required is to consume the APIs provided by the backend (ensuring consistent behaviour).

I'm also considering building a native mobile app in the future too.

[ShowMyCards]: https://showmy.cards
[Scryfall]: https://scryfall.com

`
