package test

import (
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend"
	eccTest "github.com/consensys/gnark/ecc"
)

// serializationThreshold is the number of constraints above which we don't
// don't do serialization check for the proving and verifying keys.
const serializationThreshold = 1000

// see assert.CheckCircuit for details
type profile struct {
	backends []backend.ID
	curves   []ecc.ID

	checkSerialization bool
	checkSolidity      bool
	checkProver        bool
	fuzzing            bool
	skipTestEngine     bool
}

type profile2 struct {
	backends []backend.ID
	curves   []eccTest.ID

	checkSerialization bool
	checkSolidity      bool
	checkProver        bool
	fuzzing            bool
	skipTestEngine     bool
}

var testEngineChecks = profile{
	backends: []backend.ID{},
	curves:   []ecc.ID{ecc.BN254, ecc.BLS12_381},
}

var testEngineChecks2 = profile2{
	backends: []backend.ID{},
	curves:   []eccTest.ID{eccTest.BLS12_381},
}

var constraintSolverChecks = profile{
	backends: []backend.ID{backend.GROTH16, backend.PLONK},
	curves:   []ecc.ID{ecc.BN254, ecc.BLS12_381},
}

var constraintSolverChecks2 = profile2{
	backends: []backend.ID{backend.GROTH16},
	curves:   []eccTest.ID{eccTest.BLS12_381},
}

var proverChecks = profile{
	backends:      []backend.ID{backend.GROTH16, backend.PLONK},
	curves:        []ecc.ID{ecc.BN254, ecc.BLS12_381, ecc.BW6_761},
	checkSolidity: true && SolcCheck,
	checkProver:   true,
}

var releaseChecks = profile{
	backends:           []backend.ID{backend.GROTH16, backend.PLONK},
	curves:             []ecc.ID{ecc.BN254, ecc.BLS12_381, ecc.BW6_761, ecc.BLS12_377},
	checkSolidity:      true && SolcCheck,
	checkProver:        true,
	checkSerialization: true,
	fuzzing:            true,
}

var proverChecks2 = profile2{
	backends:      []backend.ID{backend.GROTH16},
	curves:        []eccTest.ID{eccTest.BLS12_381, eccTest.BW6_767},
	checkSolidity: true && SolcCheck,
	checkProver:   true,
}

var releaseChecks2 = profile2{
	backends:           []backend.ID{backend.GROTH16},
	curves:             []eccTest.ID{eccTest.BLS12_381, eccTest.BW6_767},
	checkSolidity:      true && SolcCheck,
	checkProver:        true,
	checkSerialization: true,
	fuzzing:            true,
}
