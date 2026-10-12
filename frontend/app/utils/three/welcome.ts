import * as THREE from 'three';
import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js';
import { toonGradient, toonify } from './toon';

const modelUrl = '/models/welcome.glb';

export interface ScreenPoint {
	x: number;
	y: number;
}

// 毎フレームの画面上の位置と演出の位相
export interface WelcomeFrame {
	pebbles: ScreenPoint[];
	discs: ScreenPoint[];
	rings: ScreenPoint[];
	core: ScreenPoint;
	// 1章のタップ演出, 位相0〜1と強さ
	tap: number;
	tapWeight: number;
	// 3章の走査線, 位相0〜1と強さ
	scan: number;
	scanWeight: number;
}

export interface WelcomeScene {
	// 0〜4の章の進み具合, 小数は章の間
	setProgress: (p: number) => void;
	dispose: () => void;
}

interface Options {
	animate: boolean;
	onFrame?: (frame: WelcomeFrame) => void;
	signal?: AbortSignal;
}

// 塗りと輪郭線, palette.cssのブランド色と各色の濃色
const colors = {
	soda: [0xb9deff, 0x335b8e],
	sakura: [0xf6bed6, 0x7b3554],
	lavender: [0xddd6ff, 0x6d4eac],
	cream: [0xfff3dc, 0x8a6a3a],
} as const;
type Tone = keyof typeof colors;

function smooth(t: number): number {
	const c = Math.min(Math.max(t, 0), 1);
	return c * c * (3 - 2 * c);
}

// 区間内の進み具合
function span(p: number, from: number, to: number): number {
	return smooth((p - from) / (to - from));
}

// 章に留まっている間の強さ
function near(p: number, chapter: number): number {
	return smooth(1 - Math.abs(p - chapter) * 3);
}

const backdropShader = {
	uniforms: {
		time: { value: 0 },
		cA: { value: new THREE.Color(0xb9deff) },
		cB: { value: new THREE.Color(0xf6bed6) },
		cC: { value: new THREE.Color(0xddd6ff) },
		base: { value: new THREE.Color(0xfff9fc) },
	},
	vertexShader: `
		varying vec2 vUv;
		void main() {
			vUv = uv;
			gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
		}
	`,
	fragmentShader: `
		uniform float time;
		uniform vec3 cA;
		uniform vec3 cB;
		uniform vec3 cC;
		uniform vec3 base;
		varying vec2 vUv;
		float blob(vec2 uv, vec2 c, float r) {
			return smoothstep(r, 0.0, distance(uv, c));
		}
		void main() {
			vec2 uv = vUv;
			float t = time * 0.05;
			vec3 col = base;
			col = mix(col, cA, blob(uv, vec2(0.2 + 0.1 * sin(t * 3.0), 0.75 + 0.08 * cos(t * 2.0)), 0.75) * 0.85);
			col = mix(col, cB, blob(uv, vec2(0.85 + 0.08 * cos(t * 2.5), 0.3 + 0.1 * sin(t * 3.3)), 0.7) * 0.8);
			col = mix(col, cC, blob(uv, vec2(0.5 + 0.15 * sin(t * 1.7), 0.05 + 0.06 * cos(t * 2.2)), 0.6) * 0.7);
			gl_FragColor = vec4(col, 1.0);
		}
	`,
};

function tinted(template: THREE.Object3D, tone: Tone, gradientMap: THREE.Texture): THREE.Group {
	const holder = new THREE.Group();
	holder.add(template.clone());
	const [fill, line] = colors[tone];
	holder.traverse((o) => {
		if (o instanceof THREE.Mesh) o.material = new THREE.MeshStandardMaterial({ color: fill });
	});
	toonify(holder, gradientMap, { outline: line, outlineScale: 1.025 });
	return holder;
}

// 章ごとのカメラ位置と注視点
const shots = [
	{ pos: new THREE.Vector3(0, 0.4, 8.2), look: new THREE.Vector3(0, 0.3, 0) },
	{ pos: new THREE.Vector3(0, 0.2, 12.5), look: new THREE.Vector3(0, 0.1, 0) },
	{ pos: new THREE.Vector3(0, 0, 0), look: new THREE.Vector3(0, 0, 0) },
	{ pos: new THREE.Vector3(0, -2.9, 5.6), look: new THREE.Vector3(0, -0.4, 0) },
	{ pos: new THREE.Vector3(0, 1.2, 10.5), look: new THREE.Vector3(0, 0.3, 0) },
];

export async function mountWelcome(el: HTMLElement, opts: Options): Promise<WelcomeScene> {
	opts.signal?.throwIfAborted();
	const parts = await loadParts();
	opts.signal?.throwIfAborted();

	const renderer = new THREE.WebGLRenderer({ antialias: true });
	renderer.setPixelRatio(Math.min(window.devicePixelRatio, 1.5));
	el.appendChild(renderer.domElement);
	renderer.domElement.style.width = '100%';
	renderer.domElement.style.height = '100%';

	const scene = new THREE.Scene();
	const camera = new THREE.PerspectiveCamera(35, 1, 0.1, 100);
	scene.add(camera);
	const backdrop = new THREE.Mesh(new THREE.PlaneGeometry(1, 1), new THREE.ShaderMaterial({ ...backdropShader, depthWrite: false }));
	backdrop.position.z = -60;
	backdrop.renderOrder = -1;
	camera.add(backdrop);
	scene.add(new THREE.AmbientLight(0xffffff, 1.7));
	const sun = new THREE.DirectionalLight(0xffffff, 2.4);
	sun.position.set(4, 6, 5);
	scene.add(sun);

	const gradientMap = toonGradient();
	const part = (name: string) => parts.get(name) ?? new THREE.Mesh(new THREE.SphereGeometry(1, 32, 16));

	const core = tinted(part('Core'), 'sakura', gradientMap);
	scene.add(core);

	// 朝・昼・おやつ・夜
	const tones: Tone[] = ['soda', 'cream', 'lavender', 'sakura'];
	const pebbleNames = ['Pebble_A', 'Pebble_B', 'Pebble_C', 'Pebble_A'];
	const pebbles = tones.map((tone, i) => {
		const g = tinted(part(pebbleNames[i] ?? 'Pebble_A'), tone, gradientMap);
		scene.add(g);
		return g;
	});
	// 前回と同じ食事の複製
	const echo = tinted(part('Pebble_B'), 'cream', gradientMap);
	scene.add(echo);

	// 未計測の演出, 塗りを透明に
	const ghostIndex = 2;
	// 塗りと輪郭線の殻を消し, 面の網目だけを残す
	const ghostFades: THREE.Material[] = [];
	const ghostWires: THREE.MeshBasicMaterial[] = [];
	const ghostMeshes: THREE.Mesh[] = [];
	pebbles[ghostIndex]?.traverse((o) => {
		if (o instanceof THREE.Mesh) ghostMeshes.push(o);
	});
	for (const mesh of ghostMeshes) {
		const m = mesh.material as THREE.Material;
		m.transparent = true;
		ghostFades.push(m);
		if (m instanceof THREE.MeshToonMaterial) {
			const wireMat = new THREE.MeshBasicMaterial({ color: colors.lavender[1], wireframe: true, transparent: true, opacity: 0 });
			const wire = new THREE.Mesh(mesh.geometry, wireMat);
			wire.position.copy(mesh.position);
			wire.quaternion.copy(mesh.quaternion);
			wire.scale.copy(mesh.scale);
			mesh.parent?.add(wire);
			ghostWires.push(wireMat);
		}
	}

	const ribbon = tinted(part('Ribbon'), 'lavender', gradientMap);
	scene.add(ribbon);

	// 積み重なる記録
	const discTones: Tone[] = ['soda', 'sakura', 'lavender', 'cream', 'soda', 'sakura'];
	const discs = discTones.map((tone) => {
		const g = tinted(part('Disc'), tone, gradientMap);
		scene.add(g);
		return g;
	});

	// P・F・Cの3色のリング
	const ringTones: Tone[] = ['soda', 'sakura', 'lavender'];
	const rings = ringTones.map((tone) => {
		const g = tinted(part('Ring'), tone, gradientMap);
		scene.add(g);
		return g;
	});

	let progress = 0;
	let wide = 1;
	let width = 1;
	let height = 1;
	const clock = new THREE.Clock();
	const camPos = new THREE.Vector3();
	const camLook = new THREE.Vector3();
	const tmp = new THREE.Vector3();

	function project(v: THREE.Vector3): ScreenPoint {
		tmp.copy(v).project(camera);
		return { x: (tmp.x * 0.5 + 0.5) * width, y: (-tmp.y * 0.5 + 0.5) * height };
	}

	function layout(t: number): WelcomeFrame {
		const p = progress;
		// 0→1: 周回から弧へ, 2→3: 層へ, 3→4: リングへ
		const toArc = span(p, 0.15, 1);
		const toStack = span(p, 2.15, 3);
		const toRings = span(p, 3.15, 4);

		const coreScale = 2.5 * (1 - toArc) * (1 - toStack) + 1.05 * toRings;
		core.scale.setScalar(Math.max(coreScale, 0.001));
		core.position.set(0, 0.2 + toArc * (1 - toStack) * 0.6, 0);
		core.rotation.y = t * 0.2;
		// 呼吸
		core.scale.multiplyScalar(1 + Math.sin(t * 1.2) * 0.025 * (1 - toArc));

		pebbles.forEach((g, i) => {
			const orbit = t * 0.35 + (i * Math.PI) / 2;
			const ox = Math.cos(orbit) * 3.6 * wide;
			const oz = Math.sin(orbit) * 2.2;
			const oy = Math.sin(orbit * 1.3) * 0.5;
			const a = Math.PI * (0.88 - i * 0.25);
			const ax = Math.cos(a) * 3.6 * wide;
			const ay = Math.sin(a) * 3.6 - 1.8 + Math.sin(t * 0.9 + i) * 0.06;
			g.position.set(THREE.MathUtils.lerp(ox, ax, toArc), THREE.MathUtils.lerp(oy, ay, toArc) - toStack * 2, THREE.MathUtils.lerp(oz, 0, toArc));
			const s = THREE.MathUtils.lerp(0.75 + (i % 2) * 0.15, 0.55 + (i % 2) * 0.08, toArc) * (1 - toStack);
			g.scale.setScalar(Math.max(s, 0.001));
			g.rotation.set(t * 0.3 + i, t * 0.4 + i * 2, 0);
		});

		// 1章: 押すと同じ塊が隣に現れる
		const tapWeight = near(p, 1);
		const tap = (t % 3.2) / 3.2;
		const lunch = pebbles[1];
		if (lunch) {
			const grow = span(tap, 0.12, 0.4) * (1 - span(tap, 0.82, 0.98));
			echo.position.copy(lunch.position).add(tmp.set(0.85 * wide, 0.75, 0.2));
			echo.rotation.copy(lunch.rotation);
			echo.scale.setScalar(Math.max(lunch.scale.x * grow * tapWeight, 0.001));
		}

		const ghost = span(p, 1.4, 2) * (1 - toStack);
		for (const m of ghostFades) m.opacity = 1 - ghost;
		for (const m of ghostWires) m.opacity = ghost * 0.55;

		ribbon.position.set(0, -1.8, 0);
		ribbon.scale.set(Math.max(1.2 * wide * toArc * (1 - toStack), 0.001), Math.max(1.2 * toArc * (1 - toStack), 0.001), 1);
		ribbon.visible = toArc * (1 - toStack) > 0.001;

		discs.forEach((g, i) => {
			const appear = span(p, 2.2 + i * 0.1, 2.75 + i * 0.1) * (1 - toRings);
			g.position.set(0, -2.2 + i * 0.5 + (1 - appear) * 1.4, 0);
			g.scale.set(2.1 - i * 0.12, 1.3, 2.1 - i * 0.12).multiplyScalar(Math.max(appear, 0.001));
			g.rotation.y = i * 0.6;
		});

		rings.forEach((g, i) => {
			g.scale.setScalar(Math.max(toRings * (1.7 + i * 0.3), 0.001));
			g.position.copy(core.position);
			g.rotation.set(Math.PI / 2 + Math.sin(t * 0.4 + i * 2) * 0.5 + i * 0.7, t * (0.25 + i * 0.1), i * 1.1);
		});

		// カメラ: 章の間を補間, 2章は透明な塊へ, 3章は塔の周りを回る
		const ghostPos = pebbles[ghostIndex]?.position ?? tmp.set(0, 0, 0);
		const shot2 = shots[2];
		if (shot2) {
			shot2.pos.set(ghostPos.x * 0.6 + 0.6, ghostPos.y + 0.4, 6.8);
			shot2.look.set(ghostPos.x * 0.85 - 0.6, ghostPos.y, 0);
		}
		const from = Math.floor(Math.min(p, 3.999));
		const k = smooth(p - from);
		const a = shots[from];
		const b = shots[from + 1];
		if (a && b) {
			camPos.lerpVectors(a.pos, b.pos, k);
			camLook.lerpVectors(a.look, b.look, k);
		}
		const orbit = near(p, 3) * 0.9 + span(p, 2.4, 3) * 0.3 - span(p, 3, 3.6) * 0.3;
		const angle = orbit * (0.5 + t * 0.06);
		// 縦長は引いて全体を収める
		// 章ごとの引きの強さ, 大きな形の章ほど強く
		const pull = [1, 0.25, 0.2, 1.1, 1];
		const pw = THREE.MathUtils.lerp(pull[from] ?? 1, pull[from + 1] ?? 1, k);
		const portrait = camera.aspect < 1 ? 1 + (1 - camera.aspect) * 2.4 * pw : 1;
		// 縦長で上へ寄せる量
		const lifts = [0.48, 0.3, 0.32, 0.3, 0.45];
		const lift = THREE.MathUtils.lerp(lifts[from] ?? 0.4, lifts[from + 1] ?? 0.4, k);
		const dist = (Math.hypot(camPos.x, camPos.z) / Math.min(wide, 1)) * portrait;
		camera.position.set(Math.sin(angle) * dist + camPos.x * Math.cos(angle), camPos.y, Math.cos(angle) * dist);
		// 横長は文字の左列を避けて右へ, 縦長は下の文字を避けて上へ
		const half = Math.tan(THREE.MathUtils.degToRad(camera.fov / 2)) * camera.position.distanceTo(camLook);
		camera.lookAt(camLook);
		// 向きを保ったまま横移動
		const landscape = camera.aspect > 1.1;
		camera.position.add(tmp.set(landscape ? -half * camera.aspect * 0.38 : 0, landscape ? 0 : -half * lift, 0).applyQuaternion(camera.quaternion));

		const ringEdge = (g: THREE.Object3D) => tmp.set(1, 0, 0).applyMatrix4(g.matrixWorld);
		scene.updateMatrixWorld();
		return {
			pebbles: pebbles.map((g) => project(g.position)),
			// 画面上の右端
			discs: discs.map((g) => project(new THREE.Vector3(g.scale.x * 0.92, 0, 0).applyQuaternion(camera.quaternion).add(g.position))),
			rings: rings.map((g) => project(ringEdge(g))),
			core: project(core.position),
			tap,
			tapWeight,
			scan: (t % 4) / 4,
			scanWeight: near(p, 3),
		};
	}

	function render() {
		const t = opts.animate ? clock.getElapsedTime() : 0;
		const frame = layout(t);
		backdropShader.uniforms.time.value = t;
		renderer.render(scene, camera);
		opts.onFrame?.(frame);
	}

	function resize() {
		const { clientWidth: w, clientHeight: h } = el;
		width = w;
		height = h;
		renderer.setSize(w, h, false);
		camera.aspect = w / Math.max(h, 1);
		camera.updateProjectionMatrix();
		const bh = 2 * Math.tan(THREE.MathUtils.degToRad(camera.fov / 2)) * 60;
		backdrop.scale.set(bh * camera.aspect * 1.05, bh * 1.05, 1);
		// 縦長の画面では横方向の配置を詰める
		wide = Math.min(camera.aspect / 1.2, 1) * 0.35 + 0.65;
		render();
	}

	const observer = new ResizeObserver(resize);
	observer.observe(el);
	resize();
	if (opts.animate) renderer.setAnimationLoop(render);

	const mounted: WelcomeScene = {
		setProgress(p: number) {
			progress = Math.min(Math.max(p, 0), 4);
			if (!opts.animate) render();
		},
		dispose() {
			renderer.setAnimationLoop(null);
			observer.disconnect();
			gradientMap.dispose();
			scene.traverse((o) => {
				if (o instanceof THREE.Mesh) {
					o.geometry.dispose();
					(Array.isArray(o.material) ? o.material : [o.material]).forEach((m) => m.dispose());
				}
			});
			renderer.dispose();
			renderer.domElement.remove();
		},
	};
	opts.signal?.addEventListener('abort', () => mounted.dispose(), { once: true });
	return mounted;
}

// 名前ごとの部品, 読み込めない場合は空
async function loadParts(): Promise<Map<string, THREE.Object3D>> {
	const out = new Map<string, THREE.Object3D>();
	try {
		const gltf = await new GLTFLoader().loadAsync(modelUrl);
		for (const child of gltf.scene.children) out.set(child.name, child);
	} catch {
		// 未配置時は球で代用
	}
	return out;
}
