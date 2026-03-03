(function () {
    const canvas = document.getElementById('canvas');
    const ctx = canvas.getContext('2d');
    const W = 800, H = 600;
    const PADDLE_W = 10, PADDLE_H = 80, PADDLE_OFF = 20, BALL_SIZE = 10;
    const PADDLE_SPEED = 400; // px/s, matches server

    let ws;
    let state = null;
    let mySide = null;
    let lastDir = 0;
    let inQueue = false;

    // Client-side prediction state
    let prevState = null;
    let lastStateTime = 0;
    let prevStateTime = 0;
    let scores = { left: 0, right: 0 };
    let localPaddleY = H / 2;
    let lastFrameTime = 0;

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
                    prevState = null;
                    lastStateTime = 0;
                    prevStateTime = 0;
                    scores = { left: 0, right: 0 };
                    localPaddleY = H / 2;
                    lastFrameTime = 0;
                    document.getElementById('lobby').style.display = 'none';
                    document.getElementById('queue').style.display = 'none';
                    document.getElementById('game').style.display = 'block';
                    document.getElementById('info').textContent =
                        'You are ' + mySide + ' | vs ' + env.data.opponent;
                    break;
                case 'state':
                    handlePositionUpdate(env.data, true);
                    break;
                case 'frame':
                    handlePositionUpdate(env.data, false);
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

    function snapshotState(s) {
        return {
            ball: { x: s.ball.x, y: s.ball.y, vx: s.ball.vx, vy: s.ball.vy },
            left_paddle: { y: s.left_paddle.y },
            right_paddle: { y: s.right_paddle.y }
        };
    }

    function handlePositionUpdate(data, isFull) {
        const now = performance.now();
        if (state) prevState = snapshotState(state);
        prevStateTime = lastStateTime;
        lastStateTime = now;

        if (isFull) {
            state = data;
            scores.left = data.left_score;
            scores.right = data.right_score;
        } else {
            if (!state) {
                state = {
                    ball: { x: data.bx, y: data.by, vx: data.bvx, vy: data.bvy },
                    left_paddle: { y: data.lp },
                    right_paddle: { y: data.rp },
                    left_score: scores.left,
                    right_score: scores.right,
                    over: false,
                    winner: ''
                };
            } else {
                state.ball.x = data.bx;
                state.ball.y = data.by;
                state.ball.vx = data.bvx;
                state.ball.vy = data.bvy;
                state.left_paddle.y = data.lp;
                state.right_paddle.y = data.rp;
            }
        }

        if (mySide === 'left') {
            localPaddleY = state.left_paddle.y;
        } else {
            localPaddleY = state.right_paddle.y;
        }
    }

    // Ball prediction — single extrapolation step, clamped to field
    function predictBall(ball, dt) {
        const halfBall = BALL_SIZE / 2;
        let x = ball.x + ball.vx * dt;
        let y = ball.y + ball.vy * dt;
        // Clamp Y inside walls (mirrors one bounce)
        if (y < halfBall) y = halfBall;
        else if (y > H - halfBall) y = H - halfBall;
        return { x: x, y: y };
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

    // Rendering
    function draw() {
        const now = performance.now();
        const frameDt = lastFrameTime > 0 ? (now - lastFrameTime) / 1000 : 0;
        lastFrameTime = now;

        // Local paddle prediction — move at server speed based on current input
        if (state && frameDt > 0) {
            localPaddleY += lastDir * PADDLE_SPEED * frameDt;
            const halfPaddle = PADDLE_H / 2;
            if (localPaddleY < halfPaddle) localPaddleY = halfPaddle;
            if (localPaddleY > H - halfPaddle) localPaddleY = H - halfPaddle;
        }

        // Touch follow — send direction toward finger each frame
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

        // Predict ball position (cap dt to avoid runaway extrapolation)
        const sinceLast = lastStateTime > 0 ? Math.min((now - lastStateTime) / 1000, 0.1) : 0;
        const predicted = predictBall(state.ball, sinceLast);

        // Extrapolate opponent paddle
        let opponentY;
        const opponentServer = mySide === 'left' ? state.right_paddle.y : state.left_paddle.y;
        if (prevState && prevStateTime > 0 && lastStateTime > prevStateTime) {
            const prevOpponent = mySide === 'left' ? prevState.right_paddle.y : prevState.left_paddle.y;
            const interval = lastStateTime - prevStateTime;
            const elapsed = now - lastStateTime;
            const t = Math.min(elapsed / interval, 2);
            opponentY = opponentServer + (opponentServer - prevOpponent) * t;
            const halfPaddle = PADDLE_H / 2;
            if (opponentY < halfPaddle) opponentY = halfPaddle;
            if (opponentY > H - halfPaddle) opponentY = H - halfPaddle;
        } else {
            opponentY = opponentServer;
        }

        // Determine paddle Y values for rendering
        let leftY, rightY;
        if (mySide === 'left') {
            leftY = localPaddleY;
            rightY = opponentY;
        } else {
            leftY = opponentY;
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
        ctx.fillText(scores.left, W / 4, 60);
        ctx.fillText(scores.right, 3 * W / 4, 60);

        // Left paddle
        ctx.fillStyle = '#fff';
        ctx.fillRect(
            PADDLE_OFF,
            leftY - PADDLE_H / 2,
            PADDLE_W,
            PADDLE_H
        );

        // Right paddle
        ctx.fillRect(
            W - PADDLE_OFF - PADDLE_W,
            rightY - PADDLE_H / 2,
            PADDLE_W,
            PADDLE_H
        );

        // Ball
        ctx.fillRect(
            predicted.x - BALL_SIZE / 2,
            predicted.y - BALL_SIZE / 2,
            BALL_SIZE,
            BALL_SIZE
        );

        requestAnimationFrame(draw);
    }

    connect();
    requestAnimationFrame(draw);
})();
