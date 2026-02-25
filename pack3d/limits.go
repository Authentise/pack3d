package pack3d

// Maximum number of attempts to find a valid move (bounds + collision).
// When DoMove exhausts this many attempts without finding a valid position,
// it signals that the chosen item is stuck. Anneal uses this threshold to
// detect per-move failures.
const MAX_MOVE_ATTEMPTS = 200

// How many consecutive DoMove failures (each of MAX_MOVE_ATTEMPTS) the
// annealing loop tolerates before declaring the packing stuck.
// The actual limit is max(packItemNum * MAX_STUCK_RATIO, MAX_MOVE_ATTEMPTS)
// so that the algorithm has a chance to sample different items before
// giving up — a single stuck item should not abort the entire anneal.
const MAX_STUCK_RATIO = 3

