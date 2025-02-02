package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// ----------------------------------------------------
// 1. Constants and Vector type
// ----------------------------------------------------

// Define the window dimensions.
const (
	screenWidth  = 800
	screenHeight = 600
)

// Vector is a simple 2D vector type with helper methods.
type Vector struct {
	X, Y float64
}

// Basic vector operations.
func (v Vector) Add(u Vector) Vector {
	return Vector{v.X + u.X, v.Y + u.Y}
}

func (v Vector) Sub(u Vector) Vector {
	return Vector{v.X - u.X, v.Y - u.Y}
}

func (v Vector) Mul(s float64) Vector {
	return Vector{v.X * s, v.Y * s}
}

func (v Vector) Dot(u Vector) float64 {
	return v.X*u.X + v.Y*u.Y
}

func (v Vector) Len() float64 {
	return math.Hypot(v.X, v.Y)
}

func (v Vector) Normalize() Vector {
	l := v.Len()
	if l == 0 {
		return Vector{0, 0}
	}
	return Vector{v.X / l, v.Y / l}
}

// Perp returns a perpendicular vector (rotated 90° counterclockwise).
func (v Vector) Perp() Vector {
	return Vector{-v.Y, v.X}
}

// ----------------------------------------------------
// 2. The Game struct holds our simulation state
// ----------------------------------------------------

type Game struct {
	// Ball properties.
	ballPos Vector // Position of the ball.
	ballVel Vector // Velocity of the ball.
	ballRadius float64

	// Hexagon properties.
	hexRotation      float64 // Current rotation angle (in radians).
	hexAngularSpeed  float64 // Angular speed (radians per second).
	hexRadius        float64 // Distance from hexagon center to a vertex.

	// Pre-rendered image for the ball.
	circleImage *ebiten.Image
}

// NewGame initializes our simulation.
func NewGame() *Game {
	g := &Game{
		// Start the ball a bit above the hexagon center.
		ballPos:          Vector{X: screenWidth / 2, Y: screenHeight/2 - 150},
		// Give it an initial horizontal push.
		ballVel:          Vector{X: 100, Y: 0},
		ballRadius:       10,

		// The hexagon is centered on the screen.
		hexRotation:      0,
		hexAngularSpeed:  0.5,  // Rotate at 0.5 rad/s (adjust as desired).
		hexRadius:        200,  // Radius of the hexagon.
	}
	// Create a red circle image to represent the ball.
	g.circleImage = createCircleImage(int(g.ballRadius), color.RGBA{255, 0, 0, 255})
	return g
}

// ----------------------------------------------------
// 3. Helper: Create a filled circle image.
// ----------------------------------------------------

// createCircleImage creates an image with a filled circle of the given radius and color.
func createCircleImage(radius int, clr color.Color) *ebiten.Image {
	diameter := 2 * radius
	img := ebiten.NewImage(diameter, diameter)
	// Clear the image (transparent).
	img.Fill(color.Transparent)
	// Draw the circle pixel by pixel.
	for y := 0; y < diameter; y++ {
		for x := 0; x < diameter; x++ {
			dx := float64(x - radius)
			dy := float64(y - radius)
			if dx*dx+dy*dy <= float64(radius*radius) {
				img.Set(x, y, clr)
			}
		}
	}
	return img
}

// ----------------------------------------------------
// 4. The Update method: Physics and collision handling.
// ----------------------------------------------------

func (g *Game) Update() error {
	// We'll assume a fixed time step.
	dt := 1.0 / 60.0

	// Apply gravity to the ball (gravity pulls downward).
	gravity := 500.0 // pixels per second²
	g.ballVel.Y += gravity * dt

	// Apply a little air friction (damping) to slow the ball over time.
	airFriction := 0.99
	g.ballVel = g.ballVel.Mul(airFriction)

	// Update the ball's position.
	g.ballPos = g.ballPos.Add(g.ballVel.Mul(dt))

	// Update the hexagon’s rotation.
	g.hexRotation += g.hexAngularSpeed * dt

	// Compute the hexagon vertices (in screen coordinates).
	hexVertices := g.getHexagonVertices()

	// For each of the 6 edges, check for collision with the ball.
	// We'll use a restitution coefficient to simulate energy loss on impact.
	restitution := 0.9
	for i := 0; i < 6; i++ {
		A := hexVertices[i]
		B := hexVertices[(i+1)%6]
		// Find the closest point on the edge AB to the ball’s center.
		closest := closestPointOnSegment(A, B, g.ballPos)
		// Compute the vector from this point to the ball center.
		diff := g.ballPos.Sub(closest)
		dist := diff.Len()
		if dist < g.ballRadius {
			// --- Collision detected ---
			penetration := g.ballRadius - dist
			var normal Vector
			if dist != 0 {
				// Normal from the collision point toward the ball.
				normal = diff.Normalize()
			} else {
				// If the ball’s center is exactly on the edge, use the edge’s perpendicular.
				edge := B.Sub(A)
				normal = edge.Perp().Normalize()
			}

			// Correct the ball's position so it's no longer penetrating the wall.
			g.ballPos = g.ballPos.Add(normal.Mul(penetration))

			// To simulate a "realistic" collision with a moving wall, we
			// compute the wall’s velocity at the collision point.
			hexCenter := Vector{X: screenWidth / 2, Y: screenHeight / 2}
			r := closest.Sub(hexCenter)
			// For a rotating body, the velocity at point r is omega × r.
			// In 2D, this gives: wallVel = omega * (-r.Y, r.X)
			wallVel := r.Perp().Mul(g.hexAngularSpeed)

			// Compute the ball’s velocity relative to the moving wall.
			relVel := g.ballVel.Sub(wallVel)
			// Check if the ball is moving into the wall (dot product is negative).
			dot := relVel.Dot(normal)
			if dot < 0 {
				// Reflect the relative velocity about the collision normal.
				relVel = relVel.Sub(normal.Mul((1+restitution)*dot))
				// The new ball velocity is the reflected relative velocity plus the wall’s velocity.
				g.ballVel = relVel.Add(wallVel)
			}
		}
	}
	return nil
}

// ----------------------------------------------------
// 5. The Draw method: Rendering our scene.
// ----------------------------------------------------

func (g *Game) Draw(screen *ebiten.Image) {
	// Fill the background with a dark color.
	screen.Fill(color.RGBA{30, 30, 30, 255})

	// Draw the hexagon.
	hexVertices := g.getHexagonVertices()
	for i := 0; i < 6; i++ {
		A := hexVertices[i]
		B := hexVertices[(i+1)%6]
		// Draw a white line for each edge.
		ebitenutil.DrawLine(screen, A.X, A.Y, B.X, B.Y, color.White)
	}

	// Draw the ball.
	// We offset by the radius to center the circle image at ballPos.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-g.ballRadius, -g.ballRadius)
	op.GeoM.Translate(g.ballPos.X, g.ballPos.Y)
	screen.DrawImage(g.circleImage, op)
}

// Layout sets the window size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// ----------------------------------------------------
// 6. Utility: Compute hexagon vertices and segment collision.
// ----------------------------------------------------

// getHexagonVertices computes the 6 vertices of the rotating hexagon.
func (g *Game) getHexagonVertices() []Vector {
	center := Vector{X: screenWidth / 2, Y: screenHeight / 2}
	vertices := make([]Vector, 6)
	for i := 0; i < 6; i++ {
		angle := g.hexRotation + float64(i)*2*math.Pi/6
		vertices[i] = Vector{
			X: center.X + g.hexRadius*math.Cos(angle),
			Y: center.Y + g.hexRadius*math.Sin(angle),
		}
	}
	return vertices
}

// closestPointOnSegment returns the point on the line segment AB
// that is closest to point P.
func closestPointOnSegment(A, B, P Vector) Vector {
	AB := B.Sub(A)
	t := (P.Sub(A)).Dot(AB) / AB.Dot(AB)
	// Clamp t between 0 and 1.
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return A.Add(AB.Mul(t))
}

// ----------------------------------------------------
// 7. The main function: Run the game.
// ----------------------------------------------------

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Bouncing Ball in a Spinning Hexagon")
	game := NewGame()
	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}

