# Burner Module – User Guide

The burner module lets any user permanently remove tokens from circulation and run luck-based raffles that also destroy unclaimed prizes over time. This guide focuses on how to interact with it from a wallet/CLI point of view.

## What You Can Do
- Contribute coins to the burn pool (they will be destroyed on the next scheduled burn).
- Start a raffle for a specific denom, providing the initial prize pot and odds.
- Join an existing raffle by buying tickets and see whether you win a share of the pot.
- Query current raffles, past winners, and the history of burned coins.

## Burning Coins
Coins sent to the module are held until the periodic burn hook runs (by default once per week). At each burn, the module destroys every burnable coin it holds and records the amount per block height.

Example CLI transaction:
```bash
bzed tx burner fund-burner 1000000ubze --from mykey
```
After the next scheduled burn, the burned amount will appear in `burnedCoins` queries (see Queries below).

## Raffles
Raffles are per-denom and run for a set duration. Tickets settle a couple of blocks after purchase, so results come quickly.

### Start a Raffle
You fund the initial pot and define the rules:
- `pot`: starting prize pool in the raffle denom (must be in your balance).
- `duration`: number of days the raffle stays open (1–180). Internally this is `duration * 24` hourly epochs.
- `chances`: winning threshold between 1 and 1,000,000 (higher = better odds per ticket).
- `ratio`: decimal between 0.01 and 1.00; the winner receives `ratio * current_pot`.
- `ticket_price`: cost per ticket in the same denom.
- `denom`: the coin being raffled.

Example:
```bash
bzed tx burner start-raffle \
  --pot 10000000ubze \
  --duration 7 \
  --chances 500000 \
  --ratio 0.25 \
  --ticket-price 250000ubze \
  --denom ubze \
  --from mykey
```

### Join a Raffle
- Choose the denom of the active raffle.
- `tickets`: 1–50 tickets per transaction; each costs `ticket_price`.
- Results are evaluated two blocks after submission; each ticket is checked in consecutive blocks.
- If a ticket wins, you receive `ratio * pot` at that moment; otherwise your ticket price is added to the pot.

Example:
```bash
bzed tx burner join-raffle \
  --denom ubze \
  --tickets 3 \
  --from mykey
```

### What Happens After the Raffle Ends
- Duration is tracked in hourly epochs; after `duration` days elapse, the raffle is cleaned up.
- Remaining pot is burned; a `RaffleFinishedEvent` is emitted.
- Up to the latest 100 winners per denom are kept (older ones roll over).

## Queries (CLI / REST / gRPC)
- `bzed query burner raffles` – list active raffles.
- `bzed query burner raffle-winners --denom <denom>` – see recent winners for a denom.
- `bzed query burner all-burned-coins` – paginated list of burned amounts by block height.
- `bzed query burner params` – view current module parameters, including burn frequency.

REST routes (if using REST): `/bze/burner/raffles`, `/bze/burner/raffle_winners?denom=<denom>`, `/bze/burner/all_burned_coins`, `/bze/burner/params`.

## Governance and Permissions
- Parameter changes go through `MsgUpdateParams` and are restricted to the module authority (typically the gov module). Users normally do not send this message directly.
- All other messages (`fund-burner`, `start-raffle`, `join-raffle`) are open to any account that can cover the required funds.
