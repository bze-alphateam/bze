# TokenFactory Parameters

- **`create_denom_fee`** (`sdk.Coin`, default `25000000000ubze`): Fee charged to the creator when calling `MsgCreateDenom`. It is captured via the trade module (swapped to native if needed) and forwarded to the fee collector community-pool account.

### How It’s Used
- `MsgCreateDenom` first validates your requested `subdenom`, captures/swaps `create_denom_fee`, then creates `factory/<creator>/<subdenom>` with you as admin.

### Updating
- Only the module authority (typically governance) can update params via `MsgUpdateParams`. Supply `create_denom_fee` in the message; partial updates are rejected.
