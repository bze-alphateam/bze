# CoinTrunk Module – User Guide

CoinTrunk curates publisher-submitted articles under a set of accepted domains and lets anyone tip publishers with a small community tax.

## What You Can Do
- **Add articles** for approved domains; anonymous/unknown publishers pay a fee and are limited per period.
- **Tip publishers (“pay respect”)** in the configured denom; part of each tip goes to community pool.
- View accepted domains, articles, publishers, and publisher details via queries.

## Messages
- `MsgAddArticle`: publish an article link and optional picture URL. The article URL (and picture, if present) must be under an active accepted domain. If your publisher address is not active or unknown, the module charges the anonymous article fee and enforces the anonymous article limit for the current period.
- `MsgPayPublisherRespect`: tip a publisher in the configured denom; a percentage is redirected to the community pool, the rest goes directly to the publisher. The publisher’s “respect” counter increases by the paid amount.
- `MsgAcceptDomain` **(authority)**: add or toggle an accepted domain so only links from trusted hosts are allowed.
- `MsgSavePublisher` **(authority)**: register or update a publisher (address/name/active flag). Active publishers can post without the anonymous fee.

Example (CLI):
```bash
# Add an article (will charge the anonymous fee if publisher is not active)
bzed tx cointrunk add-article \
  "My article title" \
  "https://news.example.com/story" \
  "https://news.example.com/pic.jpg" \
  --from mypubkey

# Tip a publisher
bzed tx cointrunk pay-publisher-respect \
  <publisher-address> 5000000ubze \
  --from mykey
```

## Queries
- `bzed query cointrunk accepted-domain` – list accepted domains.
- `bzed query cointrunk articles` – list stored articles (paginated).
- `bzed query cointrunk publishers` – list publishers; `bzed query cointrunk publisher <address>` for one.
- `bzed query cointrunk anon-articles-counters` – see per-period counters used for the anonymous limit.

## Governance / Authority
- Module params and the “authority-only” messages (`accept-domain`, `save-publisher`) are reserved for the module authority (typically governance). Regular users usually only call `add-article` and `pay-publisher-respect`.

## Version History

### v8.1.0
- `SetArticle` now takes a pointer receiver, allowing the caller to observe the auto-incremented article ID
- Publisher respect tax validation now enforces strict bounds (0 < tax < 1), rejecting values >= 100%
- Removed impossible `uint64 < 0` check in `validateAnonArticleLimit`
