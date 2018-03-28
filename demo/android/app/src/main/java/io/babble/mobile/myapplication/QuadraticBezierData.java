package io.babble.mobile.myapplication;


public class QuadraticBezierData {
    public float StartX = 0f;
    public float StartY = 0f;
    public float ControlX = 0f;
    public float ControlY = 0f;
    public float EndX = 0f;
    public float EndY = 0f;

    public float LastT = 0;            //[0, 1]

    public float CurrentX = 0;
    public float CurrentY = 0;

    public int CircleBackColor = 0;
    public int CircleForeColor = 0;
    public int CircleRadius = 50;
    public String IP = "";
    public int OptimalFontSize = 12;

    public QuadraticBezierData (float startX, float startY, float controlX, float controlY, float endX, float endY, float currentX, float currentY,
                                float lastT,  int circleBackColor, int circleForeColor, int circleRadius, String iP, int optimalFontSize) {
        StartX = startX;
        StartY = startY;
        ControlX = controlX;
        ControlY = controlY;
        EndX = endX;
        EndY = endY;
        CurrentX = currentX;   //where is our point by X
        CurrentY = currentY;   //where is our point by Y
        LastT = lastT;         //Current iteration

        CircleBackColor = circleBackColor;
        CircleForeColor = circleForeColor;
        CircleRadius = Math.min(150, Math.max(50, circleRadius));     //[50, 150]
        IP = iP;
        OptimalFontSize = optimalFontSize;
    }
}
