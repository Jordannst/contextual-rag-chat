import express, { Request, Response } from 'express';
import cors from 'cors';
import uploadRoutes from './routes/uploadRoutes';

const app = express();
const PORT = 5000;

// Middleware
app.use(cors());
app.use(express.json());

// Routes
app.get('/ping', (req: Request, res: Response) => {
  res.json({ message: 'pong' });
});

// Upload routes
app.use('/api', uploadRoutes);

// Start server
app.listen(PORT, () => {
  console.log(`Server is running on port ${PORT}`);
});

