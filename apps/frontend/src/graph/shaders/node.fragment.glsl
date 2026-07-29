precision mediump float;

uniform float corruptionFactor;

void main() {
  gl_FragColor = vec4(corruptionFactor, 0.2, 0.2, 1.0);
}
