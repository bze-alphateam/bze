# CoinTrunk Parameters

- **`anon_article_limit`** (`uint64`, default `5`): Max paid (anonymous/unauthorized) articles per period. If the limit is reached, `MsgAddArticle` from inactive publishers fails until the counter resets.
- **`anon_article_cost`** (`sdk.Coin`, default `25000000000ubze`): Fee charged to inactive publishers per article; sent to the community pool during `MsgAddArticle`.
- **`publisher_respect_params.denom`** (`string`, default `ubze`): Only this denom is accepted in `MsgPayPublisherRespect`.
- **`publisher_respect_params.tax`** (`Dec`, default `0.20`): Portion of each respect payment that is redirected to the community pool; the remainder goes to the publisher.

### How They Are Used
- `MsgAddArticle` checks `anon_article_limit` and charges `anon_article_cost` when the publisher is not active/registered.
- `MsgPayPublisherRespect` enforces the `denom` and splits the payment using `tax`.

### Updating
- Params change via `MsgUpdateParams` by the module authority (usually governance). Supply all fields in the message; partial updates are rejected.
