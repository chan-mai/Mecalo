import * as THREE from 'three';
import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js';
import { toonGradient, toonify } from './toon';

const modelUrl = '/models/foods.glb';
const itemSize = 2.2;
const cellSize = 2.4;

function seeded(seed: number): () => number {
	let x = seed;
	return () => {
		x = (x * 16807) % 2147483647;
		return (x - 1) / 2147483646;
	};
}

// 食材モチーフを画面全体に淡く敷き詰める
export async function mountDecor(el: HTMLElement, opts: { animate: boolean }): Promise<() => void> {
	const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
	renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
	el.appendChild(renderer.domElement);
	renderer.domElement.style.width = '100%';
	renderer.domElement.style.height = '100%';

	const scene = new THREE.Scene();
	const camera = new THREE.PerspectiveCamera(40, 1, 0.1, 100);
	camera.position.set(0, 0, 12);
	scene.add(new THREE.AmbientLight(0xffffff, 1.5));
	const sun = new THREE.DirectionalLight(0xffffff, 2.2);
	sun.position.set(3, 5, 4);
	scene.add(sun);

	const templates = await loadTemplates();
	const gradientMap = toonGradient();
	// 色・輪郭とも淡く
	const palette = templates.map((template) => {
		const obj = template.clone();
		const size = new THREE.Box3().setFromObject(obj).getSize(new THREE.Vector3()).length();
		obj.scale.multiplyScalar(itemSize / Math.max(size, 0.001));
		toonify(obj, gradientMap, { pale: 0.45, outline: 0xb9a9e0, outlineScale: 1.05 });
		return obj;
	});

	const field = new THREE.Group();
	scene.add(field);
	let items: { obj: THREE.Object3D; base: THREE.Vector3; phase: number; spin: number }[] = [];

	// 格子の各セルへ位置をずらして配置
	function layout() {
		field.clear();
		const height = 2 * Math.tan(THREE.MathUtils.degToRad(camera.fov / 2)) * camera.position.z;
		const width = height * camera.aspect;
		const cols = Math.max(3, Math.round(width / cellSize));
		const rows = Math.max(4, Math.round(height / cellSize));
		const random = seeded(7);
		items = [];
		for (let r = 0; r < rows; r++) {
			for (let c = 0; c < cols; c++) {
				const template = palette[(r * cols + c + r) % palette.length];
				if (!template) continue;
				const obj = template.clone();
				const depth = -random() * 3;
				const base = new THREE.Vector3(
					((c + 0.5 + (random() - 0.5) * 0.6) / cols - 0.5) * width * 1.05,
					((r + 0.5 + (random() - 0.5) * 0.6) / rows - 0.5) * height * 1.05,
					depth,
				);
				// 小さめを多く, ときどき大きく
				obj.scale.multiplyScalar(0.6 + random() ** 1.6 * 1.4);
				obj.rotation.set(random() * 0.8, random() * Math.PI * 2, random() * 0.6);
				field.add(obj);
				items.push({ obj, base, phase: random() * Math.PI * 2, spin: (random() - 0.5) * 0.4 });
			}
		}
	}

	function place(t: number) {
		for (const it of items) {
			it.obj.position.set(it.base.x, it.base.y + (opts.animate ? Math.sin(t * 0.6 + it.phase) * 0.18 : 0), it.base.z);
			if (opts.animate) it.obj.rotation.y += it.spin * 0.01;
		}
	}

	function resize() {
		const { clientWidth: w, clientHeight: h } = el;
		renderer.setSize(w, h, false);
		camera.aspect = w / Math.max(h, 1);
		camera.updateProjectionMatrix();
		layout();
		place(0);
		renderer.render(scene, camera);
	}

	const observer = new ResizeObserver(resize);
	observer.observe(el);
	resize();

	const clock = new THREE.Clock();

	function frame() {
		place(clock.getElapsedTime());
		renderer.render(scene, camera);
	}

	if (opts.animate) {
		renderer.setAnimationLoop(frame);
	} else {
		frame();
	}

	return () => {
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
	};
}

// glTFがなければ
async function loadTemplates(): Promise<THREE.Object3D[]> {
	try {
		const gltf = await new GLTFLoader().loadAsync(modelUrl);
		const children = [...gltf.scene.children];
		if (children.length > 0) return children;
	} catch {
		// 未配置時は基本形状
	}
	const mat = (color: number) => new THREE.MeshStandardMaterial({ color, flatShading: true, roughness: 0.7 });
	const rice = new THREE.Mesh(new THREE.ConeGeometry(0.7, 0.9, 3), mat(0xf7f5ee));
	const tomato = new THREE.Mesh(new THREE.IcosahedronGeometry(0.5, 1), mat(0xe2513b));
	const egg = new THREE.Mesh(new THREE.SphereGeometry(0.45, 10, 8), mat(0xf3e2c3));
	egg.scale.set(1, 1.3, 1);
	const leaf = new THREE.Mesh(new THREE.TorusGeometry(0.4, 0.14, 6, 12), mat(0x5fae5b));
	return [rice, tomato, egg, leaf];
}
