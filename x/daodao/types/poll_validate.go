package types

import (
	errorsmod "cosmossdk.io/errors"
)

// Poll content / selection caps. Hardcoded (UI/UX bounds, not protocol
// policy — no Params slots).
const (
	// MinPollUserChoices is the minimum number of user-supplied choice
	// labels. Below 2 a "vote" is degenerate.
	MinPollUserChoices = 2

	// MaxPollUserChoices is the upper bound on user-supplied choices.
	// 20 is arbitrary but matches the plan; keeps tally vector small.
	MaxPollUserChoices = 20

	// MaxPollChoiceLabelLen caps each choice label's UTF-8 byte length.
	// 200 mirrors MaxProposalTitleLen.
	MaxPollChoiceLabelLen = 200

	// NotaLabel is the keeper-appended label when include_nota = true.
	// Hardcoded so UIs / off-chain consumers can detect it deterministically
	// without parsing localization.
	NotaLabel = "None of the above"
)

// ValidatePollChoices enforces the stateless rules on the user-supplied
// choice labels passed to MsgCreatePoll. Run BEFORE the keeper appends
// NOTA — the caps apply to the user input set, not the post-append set.
//
// Rules:
//   - MinPollUserChoices..MaxPollUserChoices entries.
//   - Each label is 1..MaxPollChoiceLabelLen bytes long.
//   - No duplicate labels (case-sensitive byte equality).
//   - No label equals NotaLabel (the keeper reserves that string for NOTA).
func ValidatePollChoices(choices []string) error {
	if l := len(choices); l < MinPollUserChoices || l > MaxPollUserChoices {
		return errorsmod.Wrapf(ErrInvalidPollContent,
			"choices count %d not in [%d, %d]", l, MinPollUserChoices, MaxPollUserChoices)
	}
	seen := make(map[string]struct{}, len(choices))
	for i, c := range choices {
		if l := len(c); l == 0 || l > MaxPollChoiceLabelLen {
			return errorsmod.Wrapf(ErrInvalidPollContent,
				"choices[%d] length %d not in [1, %d]", i, l, MaxPollChoiceLabelLen)
		}
		if c == NotaLabel {
			return errorsmod.Wrapf(ErrInvalidPollContent,
				"choices[%d]: label %q is reserved for NOTA (set include_nota=true instead)", i, NotaLabel)
		}
		if _, dup := seen[c]; dup {
			return errorsmod.Wrapf(ErrInvalidPollContent, "duplicate choice label %q", c)
		}
		seen[c] = struct{}{}
	}
	return nil
}

// ValidatePollSelection enforces the stateless rules on MsgVoteOnPoll's
// choice_indices selection set, given the poll's `choices` slice (post
// NOTA append) and the `max_selections` cap.
//
// Rules:
//   - 1..max_selections distinct indices (or exactly [nota_index] alone
//     when NOTA is present and chosen).
//   - Each index ∈ [0, len(choices)-1].
//   - No duplicates.
//   - NOTA exclusivity: if include_nota is true and any index equals the
//     NOTA index (= len(choices)-1 — keeper-appended), it MUST be the
//     only entry. Selecting NOTA + another choice is incoherent ("I
//     reject all options, except this one") and rejected.
//
// `includeNota` and `notaIndex` are passed explicitly so the validator
// doesn't need to inspect the Poll record itself (cleaner for table tests).
func ValidatePollSelection(choiceIndices []uint32, choicesLen int, maxSelections uint32, includeNota bool) error {
	if choicesLen <= 0 {
		return errorsmod.Wrap(ErrInvalidPollSelection, "poll has no choices")
	}
	if len(choiceIndices) == 0 {
		return errorsmod.Wrap(ErrInvalidPollSelection, "choice_indices is empty")
	}

	notaIndex := uint32(choicesLen - 1) // valid only when includeNota
	pickedNota := false
	seen := make(map[uint32]struct{}, len(choiceIndices))
	for i, idx := range choiceIndices {
		if int(idx) >= choicesLen {
			return errorsmod.Wrapf(ErrInvalidPollSelection,
				"choice_indices[%d]=%d out of range [0,%d]", i, idx, choicesLen-1)
		}
		if _, dup := seen[idx]; dup {
			return errorsmod.Wrapf(ErrInvalidPollSelection, "duplicate index %d", idx)
		}
		seen[idx] = struct{}{}
		if includeNota && idx == notaIndex {
			pickedNota = true
		}
	}

	if pickedNota {
		// NOTA exclusivity: [nota] alone is valid; everything else
		// containing nota is rejected. Allow that as the single legal
		// "reject all" signal; multi-select can't be mixed with NOTA.
		if len(choiceIndices) != 1 {
			return errorsmod.Wrap(ErrInvalidPollSelection,
				"NOTA must be the sole selection when chosen")
		}
		return nil
	}

	// Non-NOTA path: respect max_selections.
	if uint32(len(choiceIndices)) > maxSelections {
		return errorsmod.Wrapf(ErrInvalidPollSelection,
			"selected %d choices exceeds max_selections %d",
			len(choiceIndices), maxSelections)
	}
	return nil
}
