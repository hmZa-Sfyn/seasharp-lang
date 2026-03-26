namespace Geometry {

    interface IShape {
        double Area();
        double Perimeter();
    }

    class Circle : IShape {
        private double _radius;

        public Circle(double radius) {
            _radius = radius;
        }

        public double Area() {
            return 3.14159 * _radius * _radius;
        }

        public double Perimeter() {
            return 2.0 * 3.14159 * _radius;
        }

        public double Radius {
            get { return _radius; }
            set { _radius = value; }
        }
    }

    class Rectangle : IShape {
        private double _w;
        private double _h;

        public Rectangle(double w, double h) {
            _w = w;
            _h = h;
        }

        public double Area() {
            return _w * _h;
        }

        public double Perimeter() {
            return 2.0 * (_w + _h);
        }
    }
}