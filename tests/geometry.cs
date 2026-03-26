namespace Geometry
{
    public interface IShape
    {
        double Area();
        double Perimeter();
    }

    public class Circle : IShape
    {
        private double _radius;

        public Circle(double radius)
        {
            Radius = radius; // Use property for validation if added
        }

        public double Area()
        {
            return Math.PI * _radius * _radius;
        }

        public double Perimeter()
        {
            return 2 * Math.PI * _radius;
        }

        public double Radius
        {
            get => _radius;
            set => _radius = value > 0 ? value : throw new ArgumentException("Radius must be positive.");
        }
    }

    public class Rectangle : IShape
    {
        private double _width;
        private double _height;

        public Rectangle(double width, double height)
        {
            Width = width;
            Height = height;
        }

        public double Area() => _width * _height;

        public double Perimeter() => 2 * (_width + _height);

        public double Width
        {
            get => _width;
            set => _width = value > 0 ? value : throw new ArgumentException("Width must be positive.");
        }

        public double Height
        {
            get => _height;
            set => _height = value > 0 ? value : throw new ArgumentException("Height must be positive.");
        }
    }
}