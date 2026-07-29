// Scénario de charge k6 sur l'inscription — issue #91.
//
// ─────────────────────────────────────────────────────────────────────────────
// Ce que ce fichier corrige
// ─────────────────────────────────────────────────────────────────────────────
// `task test:perf` lançait `k6 run tests/perf/registration.js` sur un dossier
// qui n'existait pas. C'est la même famille de défaut que les onze gardes qui ne
// gardaient rien : un dispositif qui a l'air d'exister et n'existe pas — et
// trois documents s'appuyaient dessus, dont la grille des personas.
//
// ─────────────────────────────────────────────────────────────────────────────
// Ce que ce scénario mesure, et ce qu'il ne mesure PAS
// ─────────────────────────────────────────────────────────────────────────────
// Il mesure le chemin d'écriture le plus coûteux du socle : `POST /v1/users`
// paie un hachage Argon2id, délibérément lent. Un palier de charge sur cette
// route dit donc surtout combien de hachages la machine encaisse — c'est utile,
// à condition de savoir que c'est ça qu'on regarde.
//
// Il ne mesure PAS la base : avec la configuration livrée, tous les pilotes sont
// en mémoire. Brancher `postgres` change les chiffres du tout au tout, et c'est
// exactement pourquoi les seuils ci-dessous portent sur la FORME de la courbe
// plutôt que sur des millisecondes absolues.
//
// ⚠️ Un chiffre obtenu ici n'est comparable qu'à un autre chiffre obtenu ici,
// sur la même machine, avec la même configuration. Aucune valeur de ce fichier
// n'est une promesse de production.
//
// ─────────────────────────────────────────────────────────────────────────────
// 🔴 Le limiteur de débit est DANS le chemin mesuré
// ─────────────────────────────────────────────────────────────────────────────
// `config/http.yaml` borne le débit global à `rps: 20`, `burst: 40`. Sous la
// charge de ce scénario, le socle rend donc des 429 en quelques secondes.
//
// Mesuré à la première exécution, avant que ce fichier ne sache le détecter :
// **1 238 succès sur 1 562 202 requêtes, à 26 000 req/s**. Un chiffre flatteur
// et entièrement faux — il mesurait la vitesse à laquelle le limiteur refuse,
// pas celle à laquelle le socle inscrit.
//
// C'est la forme de faux vert la plus coûteuse : pas une erreur, un CHIFFRE.
// Un test de charge qui rend un nombre inspire confiance, et personne ne
// demande ce qui a été chronométré.
//
// Le scénario compte donc les 429 dans une métrique dédiée, avec un seuil à
// zéro. Il ne DEVINE pas la limite : une première version sondait huit
// inscriptions au démarrage, en croyant le seuil `auth_burst: 5` applicable —
// il ne l'est pas sur cette route, les huit passaient, et la sonde ne prouvait
// rien. Compter ce qui arrive vraiment est la seule mesure qui ne se trompe pas
// sur la configuration.
import http from 'k6/http'
import { check } from 'k6'
import { Rate } from 'k6/metrics'

// throttled compte la part des réponses refusées par le limiteur de débit.
//
// Une métrique plutôt qu'une sonde : elle ne peut pas se tromper sur la
// configuration réelle, puisqu'elle compte ce qui s'est produit.
const throttled = new Rate('throttled_responses')

// BASE_URL suit la même variable que les tests de bout en bout, pour qu'on ne
// pointe jamais accidentellement deux cibles différentes depuis un même poste.
const baseURL = __ENV.E2E_BASE_URL || 'http://localhost:8080'

// Un mot de passe qui satisfait le domaine : au moins douze caractères, pas
// uniquement des espaces, pas trop répétitif. S'il était refusé, le scénario
// mesurerait la vitesse à laquelle le socle rend des 422 — un résultat flatteur
// et faux, puisque le refus court-circuite le hachage.
const password = 'correct horse battery staple'

export const options = {
  // Trois paliers plutôt qu'une charge plate : ce qu'on cherche n'est pas un
  // débit maximal, c'est le moment où la latence décroche. Une charge constante
  // ne le montre jamais — elle donne une moyenne, et une moyenne cache
  // précisément le décrochage.
  stages: [
    { duration: '20s', target: 5 },
    { duration: '30s', target: 20 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    // ZÉRO échec. Ce n'est pas un seuil de confort : une inscription qui échoue
    // sous charge est un compte que quelqu'un croit avoir créé.
    checks: ['rate==1.00'],
    // Le p95 borne la queue de distribution, pas la moyenne. 2 s est large, et
    // c'est voulu : Argon2id est calibré pour coûter cher, et un seuil serré
    // ferait échouer le scénario sur une machine lente sans rien apprendre.
    http_req_failed: ['rate==0.00'],
    http_req_duration: ['p(95)<2000'],
    // UN SEUL 429 invalide la mesure. Ce n'est pas de la sévérité : dès que le
    // limiteur intervient, le chiffre obtenu n'est plus celui de l'inscription.
    //
    // `abortOnFail` arrête la course tout de suite. Sans lui, on paierait les
    // soixante secondes de paliers pour apprendre à la fin que rien de ce qui a
    // été chronométré ne comptait.
    throttled_responses: [{ threshold: 'rate==0.00', abortOnFail: true }],
  },
}

// setup s'exécute UNE fois, avant les paliers. Elle refuse tôt plutôt que de
// produire cinquante secondes de 000 : un scénario lancé contre une API éteinte
// doit le dire en une seconde, pas au dépouillement.
export function setup() {
  const probe = http.get(`${baseURL}/healthz`)
  if (probe.status !== 200) {
    throw new Error(
      `${baseURL}/healthz answered ${probe.status} — start the API first: go run ./cmd/server`,
    )
  }
}

// teardown pointe vers l'explication, il ne la répète pas.
//
// k6 nomme le seuil franchi et s'arrête là. « throttled_responses crossed »
// n'apprend rien à qui n'a pas lu ce fichier, et le remède vit dans une
// configuration non versionnée — donc introuvable par recherche.
//
// Une ligne, pas un paragraphe : un conseil affiché à chaque exécution, y
// compris réussie, cesse d'être lu au bout de trois fois. Celle-ci se contente
// de dire où lire.
//
// ⚠️ Pas de `handleSummary` avec le module `jslib.k6.io` : il ferait dépendre
// un test de charge du RÉSEAU, ce qui contredit le principe de la toolbox —
// rien d'installé sur le poste, rien à télécharger pour mesurer.
export function teardown() {
  console.log(
    'On a throttled_responses failure: raise limits.rps and limits.burst in ' +
      'config/local.yaml — see the header of tests/perf/registration.js.',
  )
}

// registerOnce envoie une inscription. Une seule définition, partagée par la
// sonde et par les paliers : deux formulations de la même requête finiraient
// par diverger, et la sonde ne prouverait plus rien sur ce qui est mesuré.
function registerOnce(address) {
  return http.post(
    `${baseURL}/v1/users`,
    JSON.stringify({ email: address, password }),
    { headers: { 'content-type': 'application/json' } },
  )
}

export default function () {
  // Une adresse unique par itération et par utilisateur virtuel : deux
  // inscriptions sur la même adresse rendraient 409, et le scénario mesurerait
  // le refus au lieu de l'écriture. Le domaine `.test` est réservé par la
  // RFC 2606, donc aucune de ces adresses ne peut exister ailleurs.
  const address = `perf-${__VU}-${__ITER}-${Date.now()}@example.test`

  const response = registerOnce(address)

  throttled.add(response.status === 429)

  check(response, {
    'status is 201': (r) => r.status === 201,
    // La réponse ne doit JAMAIS porter le condensé, même sous charge. Le
    // vérifier ici coûte trois microsecondes et garde une propriété que les
    // tests unitaires prouvent au repos — la charge est le moment où l'on
    // ajoute un cache de réponse sans y penser.
    'no digest in the body': (r) => !r.body.includes('hashed:') && !r.body.includes('$argon2'),
  })
}
