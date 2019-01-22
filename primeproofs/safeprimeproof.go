package primeproofs

import "github.com/privacybydesign/keyproof/common"
import "github.com/privacybydesign/keyproof/qspp"
import "github.com/privacybydesign/gabi/big"
import "github.com/privacybydesign/gabi/safeprime"

// A convenient safe prime is a safe prime of the form
// 2^exp - diff for small positive diff.
type ConvenientSafePrime struct {
	Exp  int
	Diff int
}

var convenientSafePrimes = []ConvenientSafePrime{
	{787, 7341},
	{836, 12077},
	{912, 7577},
	{933, 6249},
	{985, 3645},
	{1008, 3317},
	{1259, 2505},
	{1307, 4425},
	{1503, 1629},
	{1567, 3309},
	{2043, 11301},
	{2145, 429},
	{2639, 163185},
	{2659, 91209},
	{2661, 71745},
	{2705, 5445},
	{4099, 5025},
	{4682, 190265},
	{4743, 268629},
}

type SafePrimeProofStructure struct {
	N          *big.Int
	PRep       RepresentationProofStructure
	QRep       RepresentationProofStructure
	PprimeRep  RepresentationProofStructure
	QprimeRep  RepresentationProofStructure
	PPprimeRel RepresentationProofStructure
	QQprimeRel RepresentationProofStructure
	PQNRel     RepresentationProofStructure

	PprimeIsPrime PrimeProofStructure
	QprimeIsPrime PrimeProofStructure
}

type SafePrimeProof struct {
	PProof      PedersonProof
	QProof      PedersonProof
	PprimeProof PedersonProof
	QprimeProof PedersonProof
	PQNRel      *big.Int
	Challenge   *big.Int
	GroupPrime  *big.Int

	PprimeIsPrimeProof PrimeProof
	QprimeIsPrimeProof PrimeProof

	QSPPproof qspp.QuasiSafePrimeProductProof
}

type SafePrimeSecret struct {
	PQNRel           *big.Int
	PQNRelRandomizer *big.Int
}

func (s *SafePrimeSecret) GetSecret(name string) *big.Int {
	if name == "pqnrel" {
		return s.PQNRel
	}
	return nil
}

func (s *SafePrimeSecret) GetRandomizer(name string) *big.Int {
	if name == "pqnrel" {
		return s.PQNRelRandomizer
	}
	return nil
}

func (p *SafePrimeProof) GetResult(name string) *big.Int {
	if name == "pqnrel" {
		return p.PQNRel
	}
	return nil
}

func NewSafePrimeProofStructure(N *big.Int) SafePrimeProofStructure {
	var structure SafePrimeProofStructure

	structure.N = new(big.Int).Set(N)
	structure.PRep = newPedersonRepresentationProofStructure("p")
	structure.QRep = newPedersonRepresentationProofStructure("q")
	structure.PprimeRep = newPedersonRepresentationProofStructure("pprime")
	structure.QprimeRep = newPedersonRepresentationProofStructure("qprime")

	structure.PPprimeRel = RepresentationProofStructure{
		[]LhsContribution{
			LhsContribution{"p", big.NewInt(1)},
			LhsContribution{"pprime", big.NewInt(-2)},
			LhsContribution{"g", big.NewInt(-1)},
		},
		[]RhsContribution{
			RhsContribution{"h", "p_hider", 1},
			RhsContribution{"h", "pprime_hider", -2},
		},
	}

	structure.QQprimeRel = RepresentationProofStructure{
		[]LhsContribution{
			LhsContribution{"q", big.NewInt(1)},
			LhsContribution{"qprime", big.NewInt(-2)},
			LhsContribution{"g", big.NewInt(-1)},
		},
		[]RhsContribution{
			RhsContribution{"h", "q_hider", 1},
			RhsContribution{"h", "qprime_hider", -2},
		},
	}

	structure.PQNRel = RepresentationProofStructure{
		[]LhsContribution{
			LhsContribution{"g", new(big.Int).Set(N)},
		},
		[]RhsContribution{
			RhsContribution{"p", "q", 1},
			RhsContribution{"h", "pqnrel", -1},
		},
	}

	structure.PprimeIsPrime = newPrimeProofStructure("pprime", uint((N.BitLen()+1)/2))
	structure.QprimeIsPrime = newPrimeProofStructure("qprime", uint((N.BitLen()+1)/2))

	return structure
}

func (s *SafePrimeProofStructure) NumRangeProofs() int {
	return s.PprimeIsPrime.NumRangeProofs() + s.QprimeIsPrime.NumRangeProofs()
}

func findConvenientPrime(size int) *big.Int {
	for _, cp := range convenientSafePrimes {
		if int(cp.Exp) > size && int(cp.Exp)-size < 50 {
			var ret, diff big.Int
			diff.SetUint64(uint64(cp.Diff))
			ret.SetUint64(1)
			ret.Lsh(&ret, uint(cp.Exp))
			ret.Sub(&ret, &diff)
			return &ret
		}
	}
	return nil
}

func (s *SafePrimeProofStructure) BuildProof(Pprime *big.Int, Qprime *big.Int) SafePrimeProof {
	// Generate proof group
	var err error
	Follower.StepStart("Generating group prime", 0)
	primeSize := s.N.BitLen() + 2*rangeProofEpsilon + 10

	GroupPrime := findConvenientPrime(primeSize)
	if GroupPrime == nil {
		GroupPrime, err = safeprime.Generate(primeSize)
		if err != nil {
			panic(err.Error())
		}
	}
	g, gok := buildGroup(GroupPrime)
	if !gok {
		panic("Safe prime generated by gabi was not a safe prime!?")
	}
	Follower.StepDone()

	Follower.StepStart("Generating commitments", s.NumRangeProofs())
	// Build up the secrets
	PprimeSecret := newPedersonSecret(g, "pprime", Pprime)
	QprimeSecret := newPedersonSecret(g, "qprime", Qprime)
	PSecret := newPedersonSecret(g, "p", new(big.Int).Add(new(big.Int).Lsh(Pprime, 1), big.NewInt(1)))
	QSecret := newPedersonSecret(g, "q", new(big.Int).Add(new(big.Int).Lsh(Qprime, 1), big.NewInt(1)))

	PQNRelSecret := SafePrimeSecret{
		new(big.Int).Mod(new(big.Int).Mul(PSecret.hider, QSecret.secret), g.order),
		common.RandomBigInt(g.order),
	}

	// Build up bases and secrets structures
	bases := newBaseMerge(&g, &PSecret, &QSecret, &PprimeSecret, &QprimeSecret)
	secrets := newSecretMerge(&PSecret, &QSecret, &PprimeSecret, &QprimeSecret, &PQNRelSecret)

	// Build up commitment list
	var list []*big.Int
	var PprimeIsPrimeCommit PrimeProofCommit
	var QprimeIsPrimeCommit PrimeProofCommit
	var QSPPcommit qspp.QuasiSafePrimeProductCommit
	list = append(list, GroupPrime)
	list = append(list, s.N)
	list = PprimeSecret.GenerateCommitments(list)
	list = QprimeSecret.GenerateCommitments(list)
	list = PSecret.GenerateCommitments(list)
	list = QSecret.GenerateCommitments(list)
	list = s.PRep.GenerateCommitmentsFromSecrets(g, list, &bases, &secrets)
	list = s.QRep.GenerateCommitmentsFromSecrets(g, list, &bases, &secrets)
	list = s.PprimeRep.GenerateCommitmentsFromSecrets(g, list, &bases, &secrets)
	list = s.QprimeRep.GenerateCommitmentsFromSecrets(g, list, &bases, &secrets)
	list = s.PPprimeRel.GenerateCommitmentsFromSecrets(g, list, &bases, &secrets)
	list = s.QQprimeRel.GenerateCommitmentsFromSecrets(g, list, &bases, &secrets)
	list = s.PQNRel.GenerateCommitmentsFromSecrets(g, list, &bases, &secrets)
	list, PprimeIsPrimeCommit = s.PprimeIsPrime.GenerateCommitmentsFromSecrets(g, list, &bases, &secrets)
	list, QprimeIsPrimeCommit = s.QprimeIsPrime.GenerateCommitmentsFromSecrets(g, list, &bases, &secrets)
	list, QSPPcommit = qspp.QuasiSafePrimeProductBuildCommitments(list, Pprime, Qprime)
	Follower.StepDone()

	Follower.StepStart("Generating proof", 0)
	// Calculate challenge
	challenge := common.HashCommit(list)

	// Calculate proofs
	var proof SafePrimeProof
	proof.GroupPrime = GroupPrime
	proof.PQNRel = new(big.Int).Mod(
		new(big.Int).Sub(
			PQNRelSecret.PQNRelRandomizer,
			new(big.Int).Mul(
				challenge,
				PQNRelSecret.PQNRel)),
		g.order)
	proof.PProof = PSecret.BuildProof(g, challenge)
	proof.QProof = QSecret.BuildProof(g, challenge)
	proof.PprimeProof = PprimeSecret.BuildProof(g, challenge)
	proof.QprimeProof = QprimeSecret.BuildProof(g, challenge)
	proof.Challenge = challenge
	proof.PprimeIsPrimeProof = s.PprimeIsPrime.BuildProof(g, challenge, PprimeIsPrimeCommit, &secrets)
	proof.QprimeIsPrimeProof = s.QprimeIsPrime.BuildProof(g, challenge, QprimeIsPrimeCommit, &secrets)
	proof.QSPPproof = qspp.QuasiSafePrimeProductBuildProof(Pprime, Qprime, challenge, QSPPcommit)
	Follower.StepDone()

	return proof
}

func (s *SafePrimeProofStructure) VerifyProof(proof SafePrimeProof) bool {
	// Check proof structure
	Follower.StepStart("Verifying structure", 0)
	if proof.GroupPrime == nil || proof.GroupPrime.BitLen() != s.N.BitLen()+2*rangeProofEpsilon+10 {
		return false
	}
	if !proof.GroupPrime.ProbablyPrime(80) || !new(big.Int).Rsh(proof.GroupPrime, 1).ProbablyPrime(80) {
		return false
	}
	if proof.PQNRel == nil || proof.Challenge == nil {
		return false
	}
	if !proof.PProof.VerifyStructure() || !proof.QProof.VerifyStructure() {
		return false
	}
	if !proof.PprimeProof.VerifyStructure() || !proof.QprimeProof.VerifyStructure() {
		return false
	}
	if !s.PprimeIsPrime.VerifyProofStructure(proof.Challenge, proof.PprimeIsPrimeProof) ||
		!s.QprimeIsPrime.VerifyProofStructure(proof.Challenge, proof.QprimeIsPrimeProof) {
		return false
	}
	if !qspp.QuasiSafePrimeProductVerifyStructure(proof.QSPPproof) {
		return false
	}
	Follower.StepDone()

	Follower.StepStart("Rebuilding commitments", s.NumRangeProofs())

	// Rebuild group
	g, gok := buildGroup(proof.GroupPrime)
	if !gok {
		return false
	}

	// Setup names in the pederson proofs
	proof.PProof.SetName("p")
	proof.QProof.SetName("q")
	proof.PprimeProof.SetName("pprime")
	proof.QprimeProof.SetName("qprime")

	// Build up bases and secrets
	bases := newBaseMerge(&g, &proof.PProof, &proof.QProof, &proof.PprimeProof, &proof.QprimeProof)
	proofs := newProofMerge(&proof.PProof, &proof.QProof, &proof.PprimeProof, &proof.QprimeProof, &proof)

	// Build up commitment list
	var list []*big.Int
	list = append(list, proof.GroupPrime)
	list = append(list, s.N)
	list = proof.PprimeProof.GenerateCommitments(list)
	list = proof.QprimeProof.GenerateCommitments(list)
	list = proof.PProof.GenerateCommitments(list)
	list = proof.QProof.GenerateCommitments(list)
	list = s.PRep.GenerateCommitmentsFromProof(g, list, proof.Challenge, &bases, &proofs)
	list = s.QRep.GenerateCommitmentsFromProof(g, list, proof.Challenge, &bases, &proofs)
	list = s.PprimeRep.GenerateCommitmentsFromProof(g, list, proof.Challenge, &bases, &proofs)
	list = s.QprimeRep.GenerateCommitmentsFromProof(g, list, proof.Challenge, &bases, &proofs)
	list = s.PPprimeRel.GenerateCommitmentsFromProof(g, list, proof.Challenge, &bases, &proofs)
	list = s.QQprimeRel.GenerateCommitmentsFromProof(g, list, proof.Challenge, &bases, &proofs)
	list = s.PQNRel.GenerateCommitmentsFromProof(g, list, proof.Challenge, &bases, &proofs)
	list = s.PprimeIsPrime.GenerateCommitmentsFromProof(g, list, proof.Challenge, &bases, &proofs, proof.PprimeIsPrimeProof)
	list = s.QprimeIsPrime.GenerateCommitmentsFromProof(g, list, proof.Challenge, &bases, &proofs, proof.QprimeIsPrimeProof)
	list = qspp.QuasiSafePrimeProductExtractCommitments(list, proof.QSPPproof)

	Follower.StepDone()

	Follower.StepStart("Verifying proof", 0)
	defer Follower.StepDone()

	// Check challenge
	if proof.Challenge.Cmp(common.HashCommit(list)) != 0 {
		return false
	}

	// And the QSPP proof
	return qspp.QuasiSafePrimeProductVerifyProof(s.N, proof.Challenge, proof.QSPPproof)
}
