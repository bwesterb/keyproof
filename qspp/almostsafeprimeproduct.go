package qspp

import "github.com/mhe/gabi/big"

type AlmostSafePrimeProductProof struct {
	Nonce       *big.Int
	Commitments []*big.Int
	Responses   []*big.Int
}

type AlmostSafePrimeProductCommit struct {
	Nonce       *big.Int
	Commitments []*big.Int
	Logs        []*big.Int
}

func AlmostSafePrimeProductBuildCommitments(list []*big.Int, Pprime *big.Int, Qprime *big.Int) ([]*big.Int, AlmostSafePrimeProductCommit) {
	// Setup proof structure
	var commit AlmostSafePrimeProductCommit
	commit.Commitments = []*big.Int{}
	commit.Logs = []*big.Int{}

	// Calculate N and phiN
	N := new(big.Int).Mul(new(big.Int).Add(new(big.Int).Lsh(Pprime, 1), big.NewInt(1)), new(big.Int).Add(new(big.Int).Lsh(Qprime, 1), big.NewInt(1)))
	phiN := new(big.Int).Lsh(new(big.Int).Mul(Pprime, Qprime), 2)

	// Generate nonce
	nonceMax := new(big.Int).Lsh(big.NewInt(1), almostSafePrimeProductNonceSize)
	commit.Nonce = randomBigInt(nonceMax)

	for i := 0; i < almostSafePrimeProductIters; i++ {
		// Calculate base from nonce
		curc := getHashNumber(commit.Nonce, nil, i, N.BitLen())
		curc.Mod(curc, N)

		if new(big.Int).GCD(nil, nil, curc, N).Cmp(big.NewInt(1)) != 0 {
			panic("Generated number not in Z_N")
		}

		log := randomBigInt(phiN)
		com := new(big.Int).Exp(curc, log, N)
		list = append(list, com)
		commit.Commitments = append(commit.Commitments, com)
		commit.Logs = append(commit.Logs, log)
	}

	return list, commit
}

func AlmostSafePrimeProductBuildProof(Pprime *big.Int, Qprime *big.Int, challenge *big.Int, index *big.Int, commit AlmostSafePrimeProductCommit) AlmostSafePrimeProductProof {
	// Setup proof structure
	var proof AlmostSafePrimeProductProof
	proof.Nonce = commit.Nonce
	proof.Commitments = commit.Commitments
	proof.Responses = []*big.Int{}

	// Calculate useful constants
	N := new(big.Int).Mul(new(big.Int).Add(new(big.Int).Lsh(Pprime, 1), big.NewInt(1)), new(big.Int).Add(new(big.Int).Lsh(Qprime, 1), big.NewInt(1)))
	phiN := new(big.Int).Lsh(new(big.Int).Mul(Pprime, Qprime), 2)
	oddPhiN := new(big.Int).Mul(Pprime, Qprime)
	factors := []*big.Int{
		Pprime,
		Qprime,
	}

	// Calculate responses
	for i := 0; i < almostSafePrimeProductIters; i++ {
		// Derive challenge
		curc := getHashNumber(challenge, index, i, 2*N.BitLen())

		log := new(big.Int).Mod(new(big.Int).Add(commit.Logs[i], curc), phiN)

		// Calculate response
		x1 := new(big.Int).Mod(log, oddPhiN)
		x2 := new(big.Int).Sub(oddPhiN, x1)
		x3 := new(big.Int).Mod(new(big.Int).Mul(new(big.Int).ModInverse(big.NewInt(2), oddPhiN), x1), oddPhiN)
		x4 := new(big.Int).Sub(oddPhiN, x3)

		r1, ok1 := modSqrt(x1, factors)
		r2, ok2 := modSqrt(x2, factors)
		r3, ok3 := modSqrt(x3, factors)
		r4, ok4 := modSqrt(x4, factors)

		// And add the useful one
		if ok1 {
			proof.Responses = append(proof.Responses, r1)
		} else if ok2 {
			proof.Responses = append(proof.Responses, r2)
		} else if ok3 {
			proof.Responses = append(proof.Responses, r3)
		} else if ok4 {
			proof.Responses = append(proof.Responses, r4)
		} else {
			panic("none of +-x, +-x/2 are square")
		}
	}

	return proof
}

func AlmostSafePrimeProductVerifyStructure(proof AlmostSafePrimeProductProof) bool {
	if proof.Nonce == nil {
		return false
	}
	if proof.Commitments == nil || proof.Responses == nil {
		return false
	}
	if len(proof.Commitments) != almostSafePrimeProductIters || len(proof.Responses) != almostSafePrimeProductIters {
		return false
	}
	
	for _, val := range proof.Commitments {
		if val == nil {
			return false
		}
	}
	
	for _, val := range proof.Responses {
		if val == nil {
			return false
		}
	}
	
	return true
}

func AlmostSafePrimeProductExtractCommitments(list []*big.Int, proof AlmostSafePrimeProductProof) []*big.Int {
	return append(list, proof.Commitments...)
}

func AlmostSafePrimeProductVerifyProof(N *big.Int, challenge *big.Int, index *big.Int, proof AlmostSafePrimeProductProof) bool {
	// Verify N=1(mod 3), as this decreases the error prob from 9/10 to 4/5
	if new(big.Int).Mod(N, big.NewInt(3)).Cmp(big.NewInt(1)) != 0 {
		return false
	}

	// Prepare gamma
	gamma := new(big.Int).Lsh(big.NewInt(1), uint(N.BitLen()))

	// Check responses
	for i := 0; i < almostSafePrimeProductIters; i++ {
		// Generate base
		base := getHashNumber(proof.Nonce, nil, i, N.BitLen())
		base.Mod(base, N)

		// Generate challenge
		x := getHashNumber(challenge, index, i, 2*N.BitLen())
		y := new(big.Int).Mod(
			new(big.Int).Mul(
				proof.Commitments[i],
				new(big.Int).Exp(base, x, N)),
			N)

		// Verify
		yg := new(big.Int).Exp(y, gamma, N)

		t1 := new(big.Int).Exp(base, gamma, N)
		t1.Exp(t1, proof.Responses[i], N)
		t1.Exp(t1, proof.Responses[i], N)

		t2 := new(big.Int).ModInverse(t1, N)
		t3 := new(big.Int).Exp(t1, big.NewInt(2), N)
		t4 := new(big.Int).ModInverse(t3, N)

		ok1 := (t1.Cmp(yg) == 0)
		ok2 := (t2.Cmp(yg) == 0)
		ok3 := (t3.Cmp(yg) == 0)
		ok4 := (t4.Cmp(yg) == 0)

		if !ok1 && !ok2 && !ok3 && !ok4 {
			return false
		}
	}
	return true
}
