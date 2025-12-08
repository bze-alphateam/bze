# TokenFactory Parameters

- **`create_denom_fee`** (`sdk.Coin`, default `25000000000ubze`): Fee charged to the creator when calling `MsgCreateDenom`. Sent to the module’s configured destination (bank/distribution) as defined in the keeper.

### How It’s Used
- `MsgCreateDenom` first validates your requested `subdenom`, charges `create_denom_fee`, then creates `factory/<creator>/<subdenom>` with you as admin.

### Updating
- Only the module authority (typically governance) can update params via `MsgUpdateParams`. Supply `create_denom_fee` in the message; partial updates are rejected.
