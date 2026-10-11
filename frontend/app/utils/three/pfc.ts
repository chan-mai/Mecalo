import * as THREE from 'three';
import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js';
import { outlineColor, outlineOf, toonGradient } from './toon';

export interface PfcBar {
	// 摂取エネルギー比(%)
	value: number;
	// 目標範囲(%)
	min?: number;
	max?: number;
	color: string;
}

export interface PfcHandle {
	update(bars: PfcBar[]): void;
	dispose(): void;
}

const modelUrl = '/models/pfc.glb';
const fallbackMax = 70;
// 液体の最大高さ
const liquidTop = 0.963;
const bodyRadius = 0.5;
const spacing = 1.3;
// 目標上限を液体領域の8割の高さに表示
const targetLevel = 0.8;

function fillOf(bar: PfcBar): number {
	return Math.min(Math.max((bar.value / (bar.max ?? fallbackMax)) * targetLevel, 0), 1.05);
}

interface Jar {
	root: THREE.Group;
	liquid: THREE.Object3D;
	marks: THREE.Mesh[];
	current: number;
	target: number;
	phase: number;
}

export async function mountPfc(el: HTMLElement, initial: PfcBar[], opts: { animate: boolean }): Promise<PfcHandle> {
	const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
	renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
	el.appendChild(renderer.domElement);

	const scene = new THREE.Scene();
	const camera = new THREE.PerspectiveCamera(30, 1, 0.1, 50);
	camera.position.set(0, 1.9, 7.2);
	camera.lookAt(0, 0.85, 0);
	scene.add(new THREE.AmbientLight(0xffffff, 1.6));
	const light = new THREE.DirectionalLight(0xffffff, 2.2);
	light.position.set(-2, 4, 3);
	scene.add(light);

	const parts = await loadParts();
	const gradientMap = toonGradient();
	const toon = (color: THREE.ColorRepresentation, extra: THREE.MeshToonMaterialParameters = {}) => new THREE.MeshToonMaterial({ color, gradientMap, ...extra });
	const group = new THREE.Group();
	scene.add(group);

	const jars: Jar[] = initial.map((bar, i) => {
		const color = new THREE.Color(bar.color);
		const root = new THREE.Group();
		root.position.x = (i - (initial.length - 1) / 2) * spacing;

		const glass = parts.glass.clone();
		glass.material = toon(0xffffff, { transparent: true, opacity: 0.35, depthWrite: false });
		// 内側の奥の面, 輪郭線の透けを防ぐ
		const back = new THREE.Mesh(parts.glass.geometry, toon(0xf6f3ff, { side: THREE.BackSide }));
		// 目標上限までのガラスを色付き半透明に
		const zone = parts.liquid.clone();
		zone.material = toon(color, { transparent: true, opacity: 0.3, depthWrite: false });
		zone.scale.set(1.08, targetLevel, 1.08);
		const liquid = parts.liquid.clone();
		liquid.material = toon(color);
		const lid = parts.lid.clone();
		lid.material = toon(0xddd6ff);
		const ribbon = parts.ribbon.clone();
		ribbon.material = toon(color);

		// 目標範囲の下限・上限の目盛り
		const marks = [0, 1].map(() => {
			const ring = new THREE.Mesh(
				new THREE.TorusGeometry(bodyRadius + 0.012, 0.014, 6, 48),
				new THREE.MeshBasicMaterial({ color: outlineColor }),
			);
			ring.rotation.x = Math.PI / 2;
			return ring;
		});

		root.add(outlineOf(glass, 1.03), outlineOf(lid, 1.04), back, liquid, zone, glass, lid, ribbon, ...marks);
		group.add(root);
		return { root, liquid, marks, current: 0, target: 0, phase: i * 1.7 };
	});

	function update(bars: PfcBar[]) {
		bars.forEach((bar, i) => {
			const jar = jars[i];
			if (!jar) return;
			jar.target = fillOf(bar);
			const { min, max } = bar;
			const [low, high] = jar.marks;
			if (!low || !high) return;
			const hasRange = min !== undefined && max !== undefined && max > 0;
			low.visible = hasRange;
			high.visible = hasRange;
			if (hasRange) {
				low.position.y = (min / max) * targetLevel * liquidTop;
				high.position.y = targetLevel * liquidTop;
			}
		});
		if (!opts.animate) {
			jars.forEach((j) => (j.current = j.target));
			draw(0);
		}
	}

	function draw(time: number) {
		for (const j of jars) {
			j.liquid.scale.y = Math.max(j.current, 0.02);
			j.root.position.y = opts.animate ? Math.sin(time / 1500 + j.phase) * 0.04 : 0;
			j.root.rotation.z = opts.animate ? Math.sin(time / 1900 + j.phase) * 0.03 : 0;
		}
		renderer.render(scene, camera);
	}

	function resize() {
		const { clientWidth: w, clientHeight: h } = el;
		renderer.setSize(w, h, false);
		camera.aspect = w / Math.max(h, 1);
		camera.updateProjectionMatrix();
		draw(performance.now());
	}

	const observer = new ResizeObserver(resize);
	observer.observe(el);

	update(initial);
	resize();
	if (opts.animate) {
		renderer.setAnimationLoop((time) => {
			for (const j of jars) j.current += (j.target - j.current) * 0.05;
			draw(time);
		});
	}

	return {
		update,
		dispose() {
			renderer.setAnimationLoop(null);
			observer.disconnect();
			gradientMap.dispose();
			group.traverse((o) => {
				if (o instanceof THREE.Mesh) {
					o.geometry.dispose();
					(o.material as THREE.Material).dispose();
				}
			});
			renderer.dispose();
			renderer.domElement.remove();
		},
	};
}

interface Parts {
	glass: THREE.Mesh;
	liquid: THREE.Mesh;
	lid: THREE.Mesh;
	ribbon: THREE.Mesh;
}

// glTFがなければ基本形状
async function loadParts(): Promise<Parts> {
	try {
		const gltf = await new GLTFLoader().loadAsync(modelUrl);
		const find = (name: string) => gltf.scene.getObjectByName(name);
		const [glass, liquid, lid, ribbon] = ['Jar_Glass', 'Jar_Liquid', 'Jar_Lid', 'Jar_Ribbon'].map(find);
		if (glass instanceof THREE.Mesh && liquid instanceof THREE.Mesh && lid instanceof THREE.Mesh && ribbon instanceof THREE.Mesh) {
			return { glass, liquid, lid, ribbon };
		}
	} catch {
		// 未配置時
	}
	const liquid = new THREE.Mesh(new THREE.CylinderGeometry(0.45, 0.45, liquidTop, 24).translate(0, liquidTop / 2, 0));
	return {
		glass: new THREE.Mesh(new THREE.CylinderGeometry(bodyRadius, bodyRadius, 1.6, 24, 1, true).translate(0, 0.8, 0)),
		liquid,
		lid: new THREE.Mesh(new THREE.CylinderGeometry(0.35, 0.35, 0.24, 24).translate(0, 1.7, 0)),
		ribbon: new THREE.Mesh(new THREE.TorusGeometry(0.32, 0.03, 6, 32).rotateX(Math.PI / 2).translate(0, 1.34, 0)),
	};
}
