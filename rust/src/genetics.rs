use rand_core::OsRng; // requires 'getrandom' feature

use sha256::{digest};
use ecdsa::{SigningKey}
use elliptic_curve::{PublicKey, SecretKey, PrimeCurve};

use p521::elliptic_curve::{NistP521};
use p384::elliptic_curve::{NistP384};
use p224::elliptic_curve::{NistP224};


enum DNABase {
    Human(ecdsa_core::SigningKey<NistP521>),
    Bacteria(ecdsa_core::SigningKey<NistP384>),
    Virus(ecdsa_core::SigningKey<NistP224>),
}

type Protein u16;
type MHC_I PublicKey;

struct DNA {
    name: String,
    base: DNABase,
    selfProteins: Vec<Protein>,
}

struct Antigen {
	proteins: Vec<Protein>,
	signature: Signature,
}

impl DNA {
    fn new(base: DNABase, name: String) -> DNA {
        DNA {
            name: name.to_owned(),
            base: base,
        }
    }

    fn human() -> DNABase {
        DNABase::Human<ecdsa_core::SigningKey<NistP521>:random(&mut OsRng)> // Serialize with `::to_bytes()`
    }

    fn bacteria() -> DNABase {
        DNABase::Bacteria<ecdsa_core::SigningKey<NistP384>:random(&mut OsRng)> // Serialize with `::to_bytes()`
    }

    fn virus() -> DNABase {
        DNABase::Virus<ecdsa_core::SigningKey<NistP224>:random(&mut OsRng)> // Serialize with `::to_bytes()`
    }
}

fn selfProteins(base &DNABase) {

}

fn hashProteins(proteins: Vec<Protein>) -> String {
    digest(proteins)
}
