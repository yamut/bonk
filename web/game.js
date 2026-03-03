(function () {
    const canvas = document.getElementById('canvas');
    const ctx = canvas.getContext('2d');
    const W = 800, H = 600;
    const PADDLE_W = 10, PADDLE_H = 80, PADDLE_OFF = 20, BALL_SIZE = 10;
    const PADDLE_SPEED = 400; // px/s, must match server
    const HALF_PADDLE = PADDLE_H / 2;
    const HALF_BALL = BALL_SIZE / 2;

    let ws;
    let state = null;
    let mySide = null;
    let lastDir = 0;
    let inQueue = false;

    // Prediction state
    let localPaddleY = H / 2;   // our paddle, predicted locally
    let lastUpdateTime = 0;     // when we last got a server position update
    let lastFrameTime = 0;      // for delta-time between render frames

    // Connect WebSocket
    function connect() {
        const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(proto + '//' + location.host + '/ws');

        ws.onmessage = function (e) {
            const env = JSON.parse(e.data);
            switch (env.type) {
                case 'lobby':
                    document.getElementById('waiting').textContent =
                        env.data.waiting + ' player(s) waiting';
                    document.getElementById('queue-waiting').textContent =
                        env.data.waiting + ' player(s) in queue';
                    break;
                case 'start':
                    mySide = env.data.side;
                    inQueue = false;
                    state = null;
                    localPaddleY = H / 2;
                    lastUpdateTime = 0;
                    lastFrameTime = 0;
                    document.getElementById('lobby').style.display = 'none';
                    document.getElementById('queue').style.display = 'none';
                    document.getElementById('game').style.display = 'block';
                    document.getElementById('info').textContent =
                        'You are ' + mySide + ' | vs ' + env.data.opponent;
                    break;
                case 'state':
                    state = env.data;
                    onPositionUpdate();
                    break;
                case 'frame':
                    if (!state) {
                        state = {
                            ball: { x: env.data.bx, y: env.data.by, vx: env.data.bvx, vy: env.data.bvy },
                            left_paddle: { y: env.data.lp },
                            right_paddle: { y: env.data.rp },
                            left_score: 0,
                            right_score: 0,
                        };
                    } else {
                        state.ball.x = env.data.bx;
                        state.ball.y = env.data.by;
                        state.ball.vx = env.data.bvx;
                        state.ball.vy = env.data.bvy;
                        state.left_paddle.y = env.data.lp;
                        state.right_paddle.y = env.data.rp;
                    }
                    onPositionUpdate();
                    break;
                case 'over':
                    document.getElementById('game').style.display = 'none';
                    document.getElementById('game-over').style.display = 'block';
                    const won = env.data.winner === mySide;
                    document.getElementById('result').textContent =
                        won ? 'YOU WIN!' : 'YOU LOSE';
                    break;
            }
        };

        ws.onclose = function () {
            if (!state || !state.over) {
                document.getElementById('lobby').style.display = 'none';
                document.getElementById('queue').style.display = 'none';
                document.getElementById('game').style.display = 'none';
                document.getElementById('game-over').style.display = 'block';
                document.getElementById('result').textContent = 'DISCONNECTED';
            }
        };
    }

    // Called on every state/frame update from server
    function onPositionUpdate() {
        lastUpdateTime = performance.now();
        // Snap local paddle to server authoritative position
        localPaddleY = mySide === 'left' ? state.left_paddle.y : state.right_paddle.y;
    }

    // Join game
    window.join = function (mode) {
        if (inQueue) return;
        const name = document.getElementById('name').value.trim() || 'Anonymous';
        ws.send(JSON.stringify({
            type: 'join',
            data: { name: name, mode: mode }
        }));
        if (mode === 'pvp') {
            inQueue = true;
            document.getElementById('lobby').style.display = 'none';
            document.getElementById('queue').style.display = 'block';
        }
    };

    // Cancel queue
    window.cancelQueue = function () {
        ws.send(JSON.stringify({ type: 'leave', data: {} }));
        inQueue = false;
        document.getElementById('queue').style.display = 'none';
        document.getElementById('lobby').style.display = 'block';
    };

    // Input handling — send direction changes only
    const keys = {};
    document.addEventListener('keydown', function (e) {
        if (keys[e.key]) return;
        keys[e.key] = true;
        sendDirection();
    });
    document.addEventListener('keyup', function (e) {
        delete keys[e.key];
        sendDirection();
    });

    function sendDirection() {
        if (!ws || ws.readyState !== WebSocket.OPEN) return;
        let dir = 0;
        if (keys['ArrowUp'] || keys['w']) dir -= 1;
        if (keys['ArrowDown'] || keys['s']) dir += 1;
        if (dir !== lastDir) {
            lastDir = dir;
            ws.send(JSON.stringify({
                type: 'input',
                data: { direction: dir }
            }));
        }
    }

    // Touch input — follow finger
    let touchY = null;

    canvas.addEventListener('touchstart', function (e) {
        e.preventDefault();
        const rect = canvas.getBoundingClientRect();
        touchY = (e.touches[0].clientY - rect.top) / rect.height * H;
    }, { passive: false });

    canvas.addEventListener('touchmove', function (e) {
        e.preventDefault();
        const rect = canvas.getBoundingClientRect();
        touchY = (e.touches[0].clientY - rect.top) / rect.height * H;
    }, { passive: false });

    canvas.addEventListener('touchend', function (e) {
        e.preventDefault();
        touchY = null;
        if (lastDir !== 0) {
            lastDir = 0;
            ws.send(JSON.stringify({
                type: 'input',
                data: { direction: 0 }
            }));
        }
    }, { passive: false });

    function clampPaddle(y) {
        if (y < HALF_PADDLE) return HALF_PADDLE;
        if (y > H - HALF_PADDLE) return H - HALF_PADDLE;
        return y;
    }

    // Rendering
    function draw() {
        const now = performance.now();
        const frameDt = lastFrameTime > 0 ? (now - lastFrameTime) / 1000 : 0;
        lastFrameTime = now;

        // Local paddle prediction — move immediately based on input
        if (state && frameDt > 0 && frameDt < 0.1) {
            localPaddleY = clampPaddle(localPaddleY + lastDir * PADDLE_SPEED * frameDt);
        }

        // Touch follow — compare to predicted local paddle position
        if (touchY !== null && state && ws && ws.readyState === WebSocket.OPEN) {
            const deadZone = 10;
            let dir = 0;
            if (touchY < localPaddleY - deadZone) dir = -1;
            else if (touchY > localPaddleY + deadZone) dir = 1;
            if (dir !== lastDir) {
                lastDir = dir;
                ws.send(JSON.stringify({
                    type: 'input',
                    data: { direction: dir }
                }));
            }
        }

        ctx.fillStyle = '#111';
        ctx.fillRect(0, 0, W, H);

        if (!state) {
            requestAnimationFrame(draw);
            return;
        }

        // Ball: extrapolate from last server position using velocity (O(1), no loop)
        const ballDt = lastUpdateTime > 0 ? Math.min((now - lastUpdateTime) / 1000, 0.05) : 0;
        let ballX = state.ball.x + (state.ball.vx || 0) * ballDt;
        let ballY = state.ball.y + (state.ball.vy || 0) * ballDt;
        if (ballY < HALF_BALL) ballY = HALF_BALL;
        else if (ballY > H - HALF_BALL) ballY = H - HALF_BALL;

        // Paddle Y values for rendering
        let leftY, rightY;
        if (mySide === 'left') {
            leftY = localPaddleY;
            rightY = state.right_paddle.y;
        } else {
            leftY = state.left_paddle.y;
            rightY = localPaddleY;
        }

        // Center line
        ctx.setLineDash([10, 10]);
        ctx.strokeStyle = '#333';
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.moveTo(W / 2, 0);
        ctx.lineTo(W / 2, H);
        ctx.stroke();
        ctx.setLineDash([]);

        // Scores
        ctx.fillStyle = '#fff';
        ctx.font = '48px Courier New';
        ctx.textAlign = 'center';
        ctx.fillText(state.left_score || 0, W / 4, 60);
        ctx.fillText(state.right_score || 0, 3 * W / 4, 60);

        // Left paddle
        ctx.fillStyle = '#fff';
        ctx.fillRect(PADDLE_OFF, leftY - HALF_PADDLE, PADDLE_W, PADDLE_H);

        // Right paddle
        ctx.fillRect(W - PADDLE_OFF - PADDLE_W, rightY - HALF_PADDLE, PADDLE_W, PADDLE_H);

        // Ball
        ctx.fillRect(ballX - HALF_BALL, ballY - HALF_BALL, BALL_SIZE, BALL_SIZE);

        requestAnimationFrame(draw);
    }

    connect();
    requestAnimationFrame(draw);
})();
