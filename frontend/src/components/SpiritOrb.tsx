import { useRef, useMemo } from 'react';
import { Canvas, useFrame } from '@react-three/fiber';
import * as THREE from 'three';

const STAGE_COLORS: Record<number, string> = {
  1: '#a5b4fc',
  2: '#fbbf24',
  3: '#34d399',
  4: '#c084fc',
};

// Circular soft-glow sprite texture
function useGlowTexture() {
  return useMemo(() => {
    const size = 64;
    const canvas = document.createElement('canvas');
    canvas.width = size;
    canvas.height = size;
    const ctx = canvas.getContext('2d')!;
    const gradient = ctx.createRadialGradient(size / 2, size / 2, 0, size / 2, size / 2, size / 2);
    gradient.addColorStop(0, 'rgba(255, 255, 255, 1)');
    gradient.addColorStop(0.3, 'rgba(255, 255, 255, 0.6)');
    gradient.addColorStop(0.7, 'rgba(255, 255, 255, 0.1)');
    gradient.addColorStop(1, 'rgba(255, 255, 255, 0)');
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, size, size);
    const tex = new THREE.CanvasTexture(canvas);
    tex.needsUpdate = true;
    return tex;
  }, []);
}

// A shell of pulsating orb particles
function OrbShell({ count, radius, speed, color, particleSize, opacityBase }: {
  count: number; radius: number; speed: number; color: string; particleSize: number; opacityBase: number;
}) {
  const meshRef = useRef<THREE.Points>(null);
  const glowTex = useGlowTexture();

  const { positions, phases } = useMemo(() => {
    const pos = new Float32Array(count * 3);
    const ph = new Float32Array(count);
    for (let i = 0; i < count; i++) {
      const theta = Math.random() * Math.PI * 2;
      const phi = Math.acos(2 * Math.random() - 1);
      const r = radius * (0.7 + Math.random() * 0.6);
      pos[i * 3] = r * Math.sin(phi) * Math.cos(theta);
      pos[i * 3 + 1] = r * Math.sin(phi) * Math.sin(theta);
      pos[i * 3 + 2] = r * Math.cos(phi);
      ph[i] = Math.random() * Math.PI * 2; // individual phase for pulsation
    }
    return { positions: pos, phases: ph };
  }, [count, radius]);

  // Store sizes for per-particle pulsation
  const sizes = useMemo(() => {
    const s = new Float32Array(count);
    for (let i = 0; i < count; i++) {
      s[i] = particleSize * (0.5 + Math.random() * 1.0);
    }
    return s;
  }, [count, particleSize]);

  useFrame((state) => {
    if (!meshRef.current) return;
    const t = state.clock.elapsedTime;

    // Slow rotation
    meshRef.current.rotation.y = t * speed * 0.1;
    meshRef.current.rotation.x = Math.sin(t * speed * 0.05) * 0.15;

    // Global breathing
    const breathe = 1 + Math.sin(t * speed * 0.8) * 0.05;
    meshRef.current.scale.setScalar(breathe);

    // Per-particle pulsation via size attribute
    const geo = meshRef.current.geometry;
    const sizeAttr = geo.getAttribute('size');
    if (sizeAttr) {
      for (let i = 0; i < count; i++) {
        const pulse = 0.6 + Math.sin(t * 1.5 + phases[i]) * 0.4;
        (sizeAttr.array as Float32Array)[i] = sizes[i] * pulse;
      }
      sizeAttr.needsUpdate = true;
    }
  });

  return (
    <points ref={meshRef}>
      <bufferGeometry>
        <bufferAttribute attach="attributes-position" args={[positions, 3]} />
        <bufferAttribute attach="attributes-size" args={[new Float32Array(sizes), 1]} />
      </bufferGeometry>
      <pointsMaterial
        size={particleSize}
        map={glowTex}
        color={color}
        transparent
        opacity={opacityBase}
        sizeAttenuation
        blending={THREE.AdditiveBlending}
        depthWrite={false}
      />
    </points>
  );
}

// Soft glowing core sphere
function GlowCore({ stage, size = 1 }: { stage: number; size?: number }) {
  const meshRef = useRef<THREE.Mesh>(null);
  const color = STAGE_COLORS[stage] ?? STAGE_COLORS[1];

  useFrame((state) => {
    if (!meshRef.current) return;
    const t = state.clock.elapsedTime;
    const s = (0.25 + stage * 0.04) * size;
    const breathe = s + Math.sin(t * 1.5) * 0.03 * size;
    meshRef.current.scale.setScalar(breathe);
  });

  return (
    <mesh ref={meshRef}>
      <sphereGeometry args={[1, 32, 32]} />
      <meshBasicMaterial color={color} transparent opacity={0.08} />
    </mesh>
  );
}

export default function SpiritOrb({
  stage,
  size = 'md',
}: {
  stage: number;
  size?: 'sm' | 'md' | 'lg' | 'full';
}) {
  const color = STAGE_COLORS[stage] ?? STAGE_COLORS[1];

  // Fullscreen mode — multiple layered shells for depth
  if (size === 'full') {
    return (
      <div style={{ width: '100%', height: '100%', position: 'absolute', inset: 0 }}>
        <Canvas
          camera={{ position: [0, 0, 4], fov: 60 }}
          gl={{ alpha: true, antialias: true }}
          style={{ background: 'transparent' }}
        >
          <ambientLight intensity={0.3} />
          {/* Inner dense shell — bright, slow */}
          <OrbShell count={400} radius={1.2} speed={1} color={color} particleSize={0.06} opacityBase={0.5} />
          {/* Middle shell — medium density */}
          <OrbShell count={600} radius={2.2} speed={0.7} color={color} particleSize={0.05} opacityBase={0.3} />
          {/* Outer shell — sparse, large particles, fast drift */}
          <OrbShell count={300} radius={3.5} speed={0.4} color="#ffffff" particleSize={0.04} opacityBase={0.15} />
          <GlowCore stage={stage} size={2.5} />
        </Canvas>
      </div>
    );
  }

  const dims = { sm: 48, md: 120, lg: 240 }[size];
  const scale = { sm: 0.6, md: 1, lg: 1.8 }[size];
  const count = Math.floor((200 + stage * 100) * scale);

  return (
    <div style={{ width: dims, height: dims }} className="flex-shrink-0">
      <Canvas
        camera={{ position: [0, 0, 2.5], fov: 50 }}
        gl={{ alpha: true, antialias: true }}
        style={{ background: 'transparent' }}
      >
        <ambientLight intensity={0.5} />
        <OrbShell count={count} radius={1 * scale} speed={1} color={color} particleSize={0.03 * scale} opacityBase={0.7} />
        <GlowCore stage={stage} size={scale} />
      </Canvas>
    </div>
  );
}
